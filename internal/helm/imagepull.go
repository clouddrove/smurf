package helm

import "strings"

// A pod stuck on its image is the most common deploy failure, and the registry
// already says exactly why. "not found" is a tag that does not exist;
// "unauthorized" is credentials; a DNS or dial error is an unreachable
// registry. These are different problems with different fixes, and the
// difference is in the message.
//
// Asking a model to infer it produced "invalid tag or missing registry
// credentials", which is two guesses where the answer was already known.
// Classifying here means the specific cause is stated as fact, and the model
// is left to advise on a diagnosis rather than make one.

// ImagePullCause is a specific reason an image could not be pulled.
type ImagePullCause string

const (
	ImagePullTagNotFound  ImagePullCause = "image or tag does not exist in the registry"
	ImagePullUnauthorized ImagePullCause = "registry credentials are missing, wrong, or lack access"
	ImagePullRegistryDown ImagePullCause = "the registry could not be reached"
	ImagePullRateLimited  ImagePullCause = "the registry rate limit was exceeded"
	ImagePullCauseUnknown ImagePullCause = ""
)

// classifyImagePullFailure inspects a container or event message and returns
// the specific cause when the registry stated one.
//
// Order matters. An unauthorized response often also mentions the reference
// that failed, so credentials are checked before the not-found phrasing, or an
// auth problem would be reported as a missing tag and send someone to fix the
// wrong thing.
func classifyImagePullFailure(message string) ImagePullCause {
	m := strings.ToLower(message)

	switch {
	case containsAny(m, "unauthorized", "authentication required", "denied", "forbidden", "no basic auth credentials"):
		return ImagePullUnauthorized
	case containsAny(m, "toomanyrequests", "too many requests", "rate limit", "pull rate limit"):
		return ImagePullRateLimited
	case containsAny(m, "no such host", "dial tcp", "i/o timeout", "connection refused", "timeout exceeded"):
		return ImagePullRegistryDown
	case containsAny(m, "not found", "manifest unknown", "manifest for", "repository does not exist"):
		return ImagePullTagNotFound
	default:
		return ImagePullCauseUnknown
	}
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// describeImagePullFailure returns a sentence naming the image and the reason,
// or an empty string when the message says nothing definite.
//
// The image reference is included because the fix is almost always to correct
// it, and a reader should not have to go and find which image was meant.
func describeImagePullFailure(image, message string) string {
	cause := classifyImagePullFailure(message)
	if cause == ImagePullCauseUnknown {
		return ""
	}
	if image == "" {
		return string(cause)
	}
	return "image " + image + ": " + string(cause)
}

// imageFromPullMessage extracts the image reference a pull message refers to.
// Kubernetes and containerd quote it, which is what this keys on; an
// unquoted message simply yields nothing rather than a wrong guess.
func imageFromPullMessage(message string) string {
	first := strings.Index(message, `"`)
	if first < 0 {
		return ""
	}
	rest := message[first+1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// A pod can also be stuck for reasons that have nothing to do with its image,
// and those were being tolerated: an upgrade whose only problem was a pod that
// could never be scheduled still exited 0. Verified on a real cluster, where a
// pod requesting more CPU than any node has left the upgrade reporting success.
//
// The distinction that matters is not the phase but whether waiting helps.
// "Insufficient cpu" does not resolve without changing the cluster or the
// request. "ContainerCreating" resolves on its own. Only the first kind is a
// failure.

// PendingBlocker is a specific reason a pod cannot start that waiting will not
// fix.
type PendingBlocker string

const (
	PendingUnschedulable PendingBlocker = "no node can satisfy the pod's cpu, memory or placement requirements"
	PendingVolume        PendingBlocker = "its persistent volume claim cannot be satisfied"
	PendingQuota         PendingBlocker = "the namespace resource quota would be exceeded"
	PendingBlockerNone   PendingBlocker = ""
)

// classifyPendingBlocker reports why a Pending pod will stay pending, or an
// empty value when the reason looks transient.
//
// "waiting for first consumer" is deliberately not a blocker: that is the
// normal state of a WaitForFirstConsumer volume and resolves once the pod is
// scheduled, so treating it as a failure would break ordinary deployments.
func classifyPendingBlocker(message string) PendingBlocker {
	m := strings.ToLower(message)

	if strings.Contains(m, "waiting for first consumer") {
		return PendingBlockerNone
	}

	switch {
	case containsAny(m, "exceeded quota", "forbidden: exceeded"):
		return PendingQuota
	case containsAny(m, "persistentvolumeclaim", "no persistent volumes available",
		"storageclass", "unbound immediate persistentvolumeclaims", "failed to bind volumes"):
		return PendingVolume
	case containsAny(m, "failedscheduling", "unschedulable", "insufficient",
		"didn't match node selector", "didn't match pod affinity", "had taint", "had untolerated taint"):
		return PendingUnschedulable
	default:
		return PendingBlockerNone
	}
}

// describePendingBlocker returns a sentence naming why a pod is stuck, or an
// empty string when waiting may still help.
func describePendingBlocker(message string) string {
	blocker := classifyPendingBlocker(message)
	if blocker == PendingBlockerNone {
		return ""
	}
	return string(blocker)
}

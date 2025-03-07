# Bucket creation
resource "aws_s3_bucket" "my_s3_bucket"{
    bucket = "smurf-0101010101"

    tags = {
    Name = "My bucket"
    Enviroment ="Dev"
}
}
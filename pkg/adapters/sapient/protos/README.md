# SAPIENT Proto Sources

This directory vendors the official Dstl BSI Flex 335 v2 protobuf source files needed for SemOps SAPIENT descriptor
tests.

Source: <https://github.com/dstl/SAPIENT-Proto-Files>

The files are laid out under `sapient_msg/` because the official `.proto` imports use paths such as
`sapient_msg/bsi_flex_335_v2_0/alert.proto`. Keep this import layout when refreshing from upstream.

License: Apache-2.0 except where noted by Dstl. See `sapient_msg/LICENCE.txt`.

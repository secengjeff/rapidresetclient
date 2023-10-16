# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2023-10-16

- Removed mutexes from sendRequest(). Users now control how quickly a RST_STREAM frame is sent via an optional delay using the `delay` flag. 
- Minor improvement to error handling and frame counting.

## [1.0.0] - 2023-10-13

Initial release
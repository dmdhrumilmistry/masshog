# MassHog

![Masshog Logo](/.assets/images/masshog-resized.png)

**MassHog** is a command-line tool designed to help users efficiently scan multiple GitHub repositories for sensitive information using the [TruffleHog](https://github.com/trufflesecurity/trufflehog) tool. By providing a list of HTTPS URLs to repositories, MassHog simplifies the process of identifying secrets that may have been accidentally committed.

## Features

- **Batch Scanning**: Scan multiple GitHub repositories by providing a single file containing HTTPS URLs.
- **Integration with TruffleHog**: Leverage TruffleHog's powerful scanning capabilities for detecting secrets.
- **User-friendly Interface**: Easy-to-use command-line interface for quick setup and execution.
- **Configurable Options**: Customize the scan parameters for your specific needs.

## Requirements

- Go 1.23 or later
- [TruffleHog](https://github.com/trufflesecurity/trufflehog) installed on your machine

## Installation

### Using Go

* Install using `go install` command

    ```bash
    go install github.com/dmdhrumilmistry/masshog@latest
    ```

### Manual Method

* Clone the repository

    ```bash
    git clone https://github.com/dmdhrumilmistry/masshog.git
    cd masshog
    ```

* Install `masshog`

    ```bash
    go install .
    ```

## Usage

* To scan multiple GitHub repositories, create a file (e.g., `repos.txt`) that contains the HTTPS URLs of the repositories you want to scan. Each URL should be on a new line:

    ```txt
    https://github.com/owner/repo1.git
    https://github.com/owner/repo2.git
    https://github.com/owner/repo3.git
    ```

* Run MassHog with the following command

    ```bash
    masshog -f repos.txt -s state.json -o results.json
    ```

* For configurations and flags use `-h`

    ```bash
    masshog -h
    ```

## Contributing

Contributions are welcome! If you have suggestions or improvements, please create a pull request or open an issue.

* Fork the repository

* Create your feature branch 
    ```bash
    git checkout -b feature/my-feature
    ```

* Commit your changes 
    
    ```bash
    git commit -m 'Add some feature'
    ```

* Push to the branch
    
    ```bash
    git push origin feature/my-feature
    ```

* Open a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.


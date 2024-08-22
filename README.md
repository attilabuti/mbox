# mbox

> This package is based on [emersion/go-mbox](https://github.com/emersion/go-mbox).

This package offers a simple and efficient way to parse mbox files.

## Installation

To install the package, use the `go get` command:

```bash
$ go get github.com/attilabuti/mbox@latest
```

## Usage

Here's a basic example of how to use the `mbox` package:

```go
package main

import (
    "fmt"
    "io"
    "os"

    "github.com/attilabuti/mbox"
)

func main() {
    // Open the mbox file
    file, err := os.Open("path/to/your/mboxfile")
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    // Create a new mbox reader
    mboxReader := mbox.NewReader(file)

    // Iterate through messages
    for {
        message, err := mboxReader.NextMessage()
        if err == io.EOF {
            break
        }

        if err != nil {
            fmt.Println("Error reading message:", err)
            return
        }

        content, err := io.ReadAll(message)
        if err != nil {
            fmt.Println("Error reading message content:", err)
            return
        }

        fmt.Println(string(content))
    }
}
```

## Issues

Submit the [issues](https://github.com/attilabuti/mbox/issues) if you find any bug or have any suggestion.

## Contribution

Fork the [repo](https://github.com/attilabuti/mbox) and submit pull requests.

## License

This extension is licensed under the [MIT License](https://github.com/attilabuti/mbox/blob/main/LICENSE).
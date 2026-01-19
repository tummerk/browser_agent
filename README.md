# Browser Agent

Browser Agent is a Go-based service that provides browser automation capabilities using the go-rod library. The primary function is to retrieve the DOM of websites, with plans to extend functionality in the future.

## Features

- Retrieve the DOM of any website
- Navigate to specific URLs
- Manage browser instances
- Future extensibility for additional browser functions

## Architecture

The service is organized as follows:

- `internal/browser/service.go`: Main browser service implementation
- `cmd/dom-getter/main.go`: Example application demonstrating DOM retrieval

## Getting Started

### Prerequisites

- Go 1.21 or later
- Internet connection (for downloading dependencies)

### Installation

1. Clone the repository
2. Navigate to the project directory: `cd my-browser-agent`
3. Install dependencies: `go mod tidy`

### Running the Example

To run the example application that retrieves the DOM of example.com:

```bash
go run cmd/dom-getter/main.go
```

## Usage

The browser service can be integrated into your own applications:

```go
import "browser-agent/internal/browser"

ctx := context.Background()
service, err := browser.NewBrowserService(ctx)
if err != nil {
    // handle error
}
defer service.Close()

page, err := service.NavigateTo("https://example.com")
if err != nil {
    // handle error
}

dom, err := service.GetDOM(page)
if err != nil {
    // handle error
}
// Use the DOM string as needed
```

## API

### `NewBrowserService(ctx context.Context) (*BrowserService, error)`

Creates a new browser service instance.

### `GetDOM(page *rod.Page) (string, error)`

Retrieves the DOM of the current page.

### `NavigateTo(url string) (*rod.Page, error)`

Navigates to a specific URL and returns the page instance.

### `Close()`

Closes the browser service and releases resources.

### `GetPageByURL(url string) (*rod.Page, error)`

Gets an existing page by URL or creates a new one if it doesn't exist.

## Future Enhancements

- Element selection and manipulation
- Form filling and submission
- Screenshot capabilities
- JavaScript execution
- Cookie management
- Proxy support
- Headless mode configuration
# Project Layout

Wails projects have the following layout:

```
.
├── build/
│   ├── appicon.png
│   ├── darwin/
│   └── windows/
├── frontend/
├── go.mod
├── go.sum
├── main.go
└── wails.json
```

Project structure rundown

- /main.go - The main application
- /frontend/ - Frontend project files
- /build/ - Project build directory
- /build/appicon.png - The application icon
- /build/darwin/ - Mac specific project files
- /build/windows/ - Windows specific project files
- /wails.json - The project configuration
- /go.mod - Go module file
- /go.sum - Go module checksum file

The frontend directory has nothing specific to Wails and can be any frontend project of your choosing.

The build directory is used during the build process. These files may be updated to customise your builds. If files are removed from the build directory, default versions will be regenerated.

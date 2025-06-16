# README

# Developer

Para saber informações sobre a organização do projeto e desenvolvimento olhar o seguinte arquivo: [Developer.md](Developer.md)

## About

This is the official Wails React-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: <https://wails.io/docs/reference/project-config>

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on <http://localhost:34115>. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

Project Layout
Wails projects have the following layout:

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

Project structure rundown
/main.go - The main application
/frontend/ - Frontend project files
/build/ - Project build directory
/build/appicon.png - The application icon
/build/darwin/ - Mac specific project files
/build/windows/ - Windows specific project files
/wails.json - The project configuration
/go.mod - Go module file
/go.sum - Go module checksum file
The frontend directory has nothing specific to Wails and can be any frontend project of your choosing.

The build directory is used during the build process. These files may be updated to customise your builds. If files are removed from the build directory, default versions will be regenerated.

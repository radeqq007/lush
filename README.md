# Lush

A simple task runner.

# Installation

```sh
go install github.com/radeqq007/lush@latest
```

## Example Usage

```lua
local output = "bin/lush"

task("build", function()
  run("go", "build", "-o", output)
  print("Compiled.")
end)

task("format", function()
  run("go", "fmt", ".")
end)

task("release", function()
  run_task("format")
  run_task("build")
  print("Release ready: " .. output)
end)
```

```sh
lush release 
```

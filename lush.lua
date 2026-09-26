local output = "bin/lush"

task("build", function()
  run("go", "build", "-o", output)
  print("Compiled.")
end)

task("cleanup", function()
  run("rm", "-rf", "bin")
end)

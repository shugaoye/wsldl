SET YourDistroName=ubuntu18.04

:: create resources
rc /nologo res\Ubuntu\res.rc

:: compile to %YourDistroName%.exe
cl /nologo /O2 /W4 /WX /Ob2 /Oi /Oy /Gs- /GF /Gy /Tc main.c /Fe:%YourDistroName%.exe ^
  Advapi32.lib Shell32.lib shlwapi.lib wslapi.lib res\Ubuntu\res.res
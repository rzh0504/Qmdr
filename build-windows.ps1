$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$FlutterDir = Join-Path $Root "build\flutter"
$BuildPython = Join-Path $FlutterDir "build\build_python_3.12.9\python\python.exe"
$SitePackages = Join-Path $Root "build\site-packages"
$ReleaseDir = Join-Path $FlutterDir "build\windows\x64\runner\Release"
$DistDir = Join-Path $Root "build\windows"
$AppIcon = Join-Path $Root "assets\qmdr.ico"
$FlutterWindowsIcon = Join-Path $FlutterDir "windows\runner\resources\app_icon.ico"
$FlutterWindowsBuildDir = Join-Path $FlutterDir "build\windows"
$FlutterMainDart = Join-Path $FlutterDir "lib\main.dart"

$env:PYTHONUTF8 = "1"
$env:PYTHONIOENCODING = "utf-8"

$FlutterBat = $env:FLUTTER_BIN
if ($FlutterBat -and -not (Test-Path -LiteralPath $FlutterBat)) {
    $FlutterBat = $null
}
if (-not $FlutterBat) {
    $DefaultFlutter = "D:\Dev\Flutter\flutter\bin\flutter.BAT"
    if (Test-Path -LiteralPath $DefaultFlutter) {
        $FlutterBat = $DefaultFlutter
    } else {
        $FlutterCommand = Get-Command flutter -ErrorAction SilentlyContinue
        if ($FlutterCommand) {
            $FlutterBat = $FlutterCommand.Source
        }
    }
}

if (-not $FlutterBat -or -not (Test-Path -LiteralPath $FlutterBat)) {
    throw "Flutter executable not found. Set FLUTTER_BIN to flutter.bat."
}

if (-not (Test-Path -LiteralPath (Join-Path $FlutterDir "pubspec.yaml"))) {
    Write-Host "Creating Flet Flutter shell..."
    uv run flet build windows --yes --no-rich-output --skip-flutter-doctor -vv
}

if (-not (Test-Path -LiteralPath $BuildPython)) {
    Write-Host "Preparing embedded Python with Flet..."
    uv run flet build windows --yes --no-rich-output --skip-flutter-doctor -vv
}

if (Test-Path -LiteralPath $AppIcon) {
    Write-Host "Applying Windows executable icon..."
    Copy-Item -LiteralPath $AppIcon -Destination $FlutterWindowsIcon -Force
    if (Test-Path -LiteralPath $FlutterWindowsBuildDir) {
        Remove-Item -LiteralPath $FlutterWindowsBuildDir -Recurse -Force
    }
}

if (Test-Path -LiteralPath $FlutterMainDart) {
    Write-Host "Applying native startup loading screen..."
    $MainDart = Get-Content -LiteralPath $FlutterMainDart -Raw
    $MainDart = $MainDart.Replace('final showAppBootScreen = bool.tryParse("False".toLowerCase()) ?? false;', 'final showAppBootScreen = bool.tryParse("True".toLowerCase()) ?? false;')
    $MainDart = $MainDart.Replace("const appBootScreenMessage = 'Preparing the app for its first launch…';", "const appBootScreenMessage = '';")
    $MainDart = $MainDart.Replace('final showAppStartupScreen = bool.tryParse("False".toLowerCase()) ?? false;', 'final showAppStartupScreen = bool.tryParse("True".toLowerCase()) ?? false;')
    $MainDart = $MainDart.Replace("const appStartupScreenMessage = 'Getting things ready…';", "const appStartupScreenMessage = '';")
    $SimpleBootScreen = @'
class BootScreen extends StatelessWidget {
  const BootScreen({
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return const MaterialApp(
      debugShowCheckedModeBanner: false,
      home: Scaffold(
        backgroundColor: Color(0xFFF5F7FB),
        body: Center(
          child: SizedBox(
            width: 32,
            height: 32,
            child: CircularProgressIndicator(strokeWidth: 3),
          ),
        ),
      ),
    );
  }
}
'@
    $MainDart = [System.Text.RegularExpressions.Regex]::Replace(
        $MainDart,
        'class BootScreen extends StatelessWidget \{.*?\r?\n\}\r?\n\r?\nclass BlankScreen',
        $SimpleBootScreen + "`r`n`r`nclass BlankScreen",
        [System.Text.RegularExpressions.RegexOptions]::Singleline
    )
    Set-Content -LiteralPath $FlutterMainDart -Value $MainDart -Encoding UTF8
}

Write-Host "Installing Python dependencies to $SitePackages..."
if (Test-Path -LiteralPath $SitePackages) {
    Remove-Item -LiteralPath $SitePackages -Recurse -Force
}

& $BuildPython -m pip install `
    --upgrade `
    --disable-pip-version-check `
    --extra-index-url https://pypi.flet.dev `
    --target $SitePackages `
    "aiofiles >=25.1.0" `
    "aiohttp >=3.13.5" `
    "certifi >=2026.4.22" `
    "flet <0.86.0, >=0.85.0" `
    "mutagen >=1.47.0" `
    "qqmusic-api-python==0.3.6"

if ($LASTEXITCODE -ne 0) {
    throw "pip install failed with exit code $LASTEXITCODE."
}

Write-Host "Packaging Python app..."
$env:SERIOUS_PYTHON_SITE_PACKAGES = $SitePackages
$env:SERIOUS_PYTHON_FLUTTER_PACKAGES = Join-Path $Root "build\flutter-packages-temp"

$DartBat = Join-Path (Split-Path -Parent $FlutterBat) "dart.BAT"
if (-not (Test-Path -LiteralPath $DartBat)) {
    $DartCommand = Get-Command dart -ErrorAction SilentlyContinue
    if ($DartCommand) {
        $DartBat = $DartCommand.Source
    }
}

if (-not $DartBat -or -not (Test-Path -LiteralPath $DartBat)) {
    throw "Dart executable not found. Ensure Flutter's bin directory is available."
}

Push-Location $FlutterDir
try {
    & $FlutterBat pub get
    if ($LASTEXITCODE -ne 0) {
        throw "flutter pub get failed with exit code $LASTEXITCODE."
    }

    & $DartBat run --suppress-analytics serious_python:main package $Root --platform Windows --exclude build --cleanup-packages --skip-site-packages --verbose
    if ($LASTEXITCODE -ne 0) {
        throw "serious_python package failed with exit code $LASTEXITCODE."
    }

    Write-Host "Building Windows executable..."
    $env:SERIOUS_PYTHON_SITE_PACKAGES = $SitePackages
    & $FlutterBat build windows
    if ($LASTEXITCODE -ne 0) {
        throw "flutter build windows failed with exit code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}

Write-Host "Copying release files to $DistDir..."
if (Test-Path -LiteralPath $DistDir) {
    Remove-Item -LiteralPath $DistDir -Recurse -Force
}
New-Item -ItemType Directory -Path $DistDir | Out-Null
Get-ChildItem -LiteralPath $ReleaseDir | Copy-Item -Destination $DistDir -Recurse -Force

Write-Host "Build complete: $(Join-Path $DistDir 'qmdr.exe')"

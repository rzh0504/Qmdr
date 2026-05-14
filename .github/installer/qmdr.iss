#ifndef AppVersion
#define AppVersion "0.1.0"
#endif

#ifndef RootDir
#define RootDir "..\.."
#endif

#ifndef SourceDir
#define SourceDir "..\..\build\windows"
#endif

#ifndef OutputDir
#define OutputDir "..\..\dist"
#endif

[Setup]
AppId={{3D3F9454-9586-45F1-8A67-E9D5278F644D}
AppName=Qmdr
AppVersion={#AppVersion}
AppPublisher=Qmdr
DefaultDirName={autopf}\Qmdr
DefaultGroupName=Qmdr
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename=Qmdr-Setup-{#AppVersion}
SetupIconFile={#RootDir}\assets\qmdr.ico
UninstallDisplayIcon={app}\qmdr.exe
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: unchecked

[Files]
Source: "{#SourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\Qmdr"; Filename: "{app}\qmdr.exe"
Name: "{autodesktop}\Qmdr"; Filename: "{app}\qmdr.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\qmdr.exe"; Description: "Launch Qmdr"; Flags: nowait postinstall skipifsilent

; XJTU Housing Genius — Inno Setup Script
; v1.0

#define MyAppName "XJTU Housing Genius"
#define MyAppVersion "1.0"
#define MyAppPublisher "Hz"
#define MyAppExeName "xjtu_housing_genius.exe"

[Setup]
AppId={{F8C3A2D1-6B4E-4F9A-8C3D-1E5F7A9B2C6D}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\XJTUHousingGenius
UsePreviousAppDir=yes
UninstallDisplayIcon={app}\{#MyAppExeName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=XJTU-Housing-Genius-Setup-v{#MyAppVersion}
SolidCompression=yes
Compression=lzma2/ultra
SetupIconFile=..\frontend\windows\runner\resources\app_icon.ico
WizardStyle=modern
PrivilegesRequiredOverridesAllowed=dialog

[Languages]
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
; Flutter app
Source: "..\frontend\build\windows\x64\runner\Release\xjtu_housing_genius.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\frontend\build\windows\x64\runner\Release\flutter_windows.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\frontend\build\windows\x64\runner\Release\data\*"; DestDir: "{app}\data"; Flags: ignoreversion recursesubdirs createallsubdirs
; Go backend
Source: "..\backend\xjtu-housing-genius.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[UninstallDelete]
Type: filesandordirs; Name: "{app}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
function InitializeUninstall(): Boolean;
var ResultCode: Integer;
begin
  Exec('taskkill', '/f /im xjtu_housing_genius.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('taskkill', '/f /im xjtu-housing-genius.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := True;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var ResultCode: Integer;
begin
  Exec('taskkill', '/f /im xjtu_housing_genius.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('taskkill', '/f /im xjtu-housing-genius.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := '';
end;

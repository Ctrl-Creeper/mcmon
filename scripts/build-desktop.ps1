param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$WailsArgs
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2 build -clean -webview2 download @WailsArgs

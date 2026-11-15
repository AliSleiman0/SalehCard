<#
.SYNOPSIS
  Grants one RBAC permission area (default: complaints) to custom admin roles in PROD, with no admin
  password needed.

.DESCRIPTION
  Uses YOUR Azure CLI login instead of an admin account:
    1. reads MONGO_URI from the salehcard-api App Service settings (never printed),
    2. opens a temporary Cosmos firewall rule for this PC's public IP,
    3. runs `dbtool roles` and a dry run of `dbtool role-grant`, shows the plan,
    4. applies it only after you type "yes" (each changed role gets an admin_audit_log row,
       action role.update, actor "dbtool (ops CLI)"),
    5. always deletes the firewall rule again, even on failure.

  Needs: `az login` done (an account with access to the salehcard-prod resource group) and Go installed.
  Role holders get the new access on their next token refresh or login. Super admins (admins with no
  custom role) already have every permission.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deploy\grant-role-permission.ps1
.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deploy\grant-role-permission.ps1 -Level view -DryRun
#>
param(
  [string]$Domain = 'complaints',
  [ValidateSet('view', 'manage')][string]$Level = 'manage',
  [string]$RoleId = 'all',        # a role id from the listing, or "all"
  [switch]$DryRun,
  [string]$ResourceGroup = 'salehcard-prod',
  [string]$WebApp = 'salehcard-api',
  [string]$Cluster = 'docdb-cluster-20260626-2230',
  [string]$LocalUri = ''          # testing only: run against this Mongo URI, skip Azure + firewall
)

$ErrorActionPreference = 'Stop'
$apiDir = Join-Path $PSScriptRoot '..\api'
if (-not (Test-Path (Join-Path $apiDir 'cmd\dbtool'))) { throw "Can't find api\cmd\dbtool next to this script." }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go is not on PATH.' }

function Invoke-Dbtool {
  param([string[]]$DbArgs)
  Push-Location $apiDir
  # dbtool logs to stderr; Windows PowerShell 5.1 would turn that into a terminating error under
  # 'Stop', so judge success by the exit code instead.
  $prev = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try {
    & go run ./cmd/dbtool @DbArgs 2>&1 | ForEach-Object { "$_" }
    if ($LASTEXITCODE -ne 0) { throw "dbtool $($DbArgs -join ' ') failed (exit $LASTEXITCODE)." }
  } finally {
    $ErrorActionPreference = $prev
    Pop-Location
  }
}

$ruleUrl = $null
$oldUri = $env:MONGO_URI
$oldDb = $env:DB_NAME
try {
  if ($LocalUri) {
    $env:MONGO_URI = $LocalUri
    if (-not $env:DB_NAME) { $env:DB_NAME = 'salehcard' }
    Write-Host "Using local Mongo (test mode)." -ForegroundColor Yellow
  } else {
    if (-not (Get-Command az -ErrorAction SilentlyContinue)) { throw 'Azure CLI (az) is not on PATH.' }
    $sub = (& az account show --query id -o tsv)
    if (-not $sub) { throw 'Not logged in to Azure. Run: az login' }
    $sub = $sub.Trim()

    Write-Host 'Reading the prod database connection from the App Service settings...'
    $uri = (& az webapp config appsettings list -g $ResourceGroup -n $WebApp --query "[?name=='MONGO_URI'].value | [0]" -o tsv)
    if (-not $uri) { throw "Couldn't read MONGO_URI from $WebApp (does your Azure account have access to $ResourceGroup?)." }
    $db = (& az webapp config appsettings list -g $ResourceGroup -n $WebApp --query "[?name=='DB_NAME'].value | [0]" -o tsv)
    $env:MONGO_URI = $uri.Trim()
    $env:DB_NAME = if ($db) { $db.Trim() } else { 'salehcard' }
    $uri = $null

    $ip = (Invoke-RestMethod -Uri 'https://api.ipify.org').Trim()
    $rule = 'grantperm-' + (Get-Date -Format 'yyyyMMddHHmm')
    $ruleUrl = "https://management.azure.com/subscriptions/$sub/resourceGroups/$ResourceGroup/providers/Microsoft.DocumentDB/mongoClusters/$Cluster/firewallRules/${rule}?api-version=2024-07-01"
    Write-Host "Opening a temporary database firewall rule for $ip ($rule)..."
    $bodyFile = [IO.Path]::GetTempFileName()
    Set-Content -Path $bodyFile -Value ('{"properties":{"startIpAddress":"' + $ip + '","endIpAddress":"' + $ip + '"}}') -Encoding ascii
    & az rest --method put --url $ruleUrl --body "@$bodyFile" --output none
    Remove-Item $bodyFile -ErrorAction SilentlyContinue
    if ($LASTEXITCODE -ne 0) { $ruleUrl = $null; throw 'Creating the firewall rule failed.' }
    for ($i = 0; $i -lt 30; $i++) {
      $state = (& az rest --method get --url $ruleUrl --query 'properties.provisioningState' -o tsv)
      if ("$state".Trim() -eq 'Succeeded') { break }
      Start-Sleep -Seconds 5
    }
    Write-Host "Firewall rule state: $state"
  }

  Write-Host ''
  Write-Host '== Custom roles ==' -ForegroundColor Cyan
  Invoke-Dbtool @('roles')
  Write-Host ''
  Write-Host "== Plan: grant $Domain ($Level) to $RoleId ==" -ForegroundColor Cyan
  Invoke-Dbtool @('role-grant', $Domain, $Level, $RoleId)

  if ($DryRun) {
    Write-Host 'Dry run only: nothing changed.' -ForegroundColor Yellow
  } else {
    $confirm = Read-Host "Apply this to the database? Type 'yes'"
    if ($confirm -eq 'yes') {
      Invoke-Dbtool @('role-grant', $Domain, $Level, $RoleId, '--apply')
    } else {
      Write-Host 'Cancelled: nothing changed.'
    }
  }
} finally {
  $env:MONGO_URI = $oldUri
  $env:DB_NAME = $oldDb
  if ($ruleUrl) {
    Write-Host 'Removing the temporary firewall rule...'
    & az rest --method delete --url $ruleUrl --output none
    if ($LASTEXITCODE -ne 0) { Write-Host "Could not delete the firewall rule. Remove it in the Azure portal: $ruleUrl" -ForegroundColor Red }
  }
}

<#
.SYNOPSIS
  Grants one RBAC permission area (default: complaints) to custom admin roles, through the admin API.

.DESCRIPTION
  Signs in as a SUPER ADMIN (role edits are super-admin-only), lists the custom roles with what they hold
  today, asks which roles to update and at which level (view or manage), shows the plan, and applies it only
  after you type "yes". Each change goes through PUT /api/admin/roles/{id}, so it is validated and audited
  ("role.update"), the same as using admin -> Roles.

  Role holders get the new permission on their next token refresh or login. Super admins (admins with no
  custom role) already have every permission and are not listed.

  Nothing is stored: the password is read as a secure string and the access token lives only in memory.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deploy\grant-role-permission.ps1
.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deploy\grant-role-permission.ps1 -Domain complaints -DryRun
#>
param(
  [string]$ApiBase = 'https://salehcard-api.azurewebsites.net',
  [string]$Domain = 'complaints',
  [string]$Email = '',
  [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

function Invoke-Api {
  param([string]$Method, [string]$Path, $Body = $null, [string]$Token = '')
  $headers = @{ 'Accept' = 'application/json' }
  if ($Token) { $headers['Authorization'] = "Bearer $Token" }
  $req = @{ Method = $Method; Uri = "$ApiBase$Path"; Headers = $headers; ContentType = 'application/json' }
  if ($null -ne $Body) { $req['Body'] = ($Body | ConvertTo-Json -Depth 5 -Compress) }
  try {
    $res = Invoke-RestMethod @req
  } catch {
    $detail = $_.ErrorDetails.Message
    if (-not $detail) { $detail = $_.Exception.Message }
    throw "$Method $Path failed: $detail"
  }
  if (-not $res.success) { throw "$Method $Path failed: $($res.error.message)" }
  return $res.data
}

# ---- sign in (handles the admin SMS second factor when it is on) ------------------------------
if (-not $Email) {
  $Email = Read-Host 'Super-admin email [admin@salehcard.com]'
  if (-not $Email) { $Email = 'admin@salehcard.com' }
}
$secure = Read-Host 'Password' -AsSecureString
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
$auth = Invoke-Api -Method POST -Path '/api/v1/auth/login' -Body @{ email = $Email; password = $plain }
$plain = $null
if ($auth.twoFactorRequired) {
  $code = Read-Host "SMS code sent to $($auth.phoneHint)"
  $auth = Invoke-Api -Method POST -Path '/api/v1/auth/2fa/verify' -Body @{ pendingToken = $auth.pendingToken; code = $code.Trim() }
}
$token = $auth.accessToken
if (-not $token) { throw 'Sign-in returned no access token.' }
if (-not ($auth.user.permissions -contains '*')) {
  throw "$Email is not a super admin. Only super admins can edit roles."
}
Write-Host "Signed in as $Email (super admin)." -ForegroundColor Green

# ---- list roles ----------------------------------------------------------------------------------
$catalog = Invoke-Api -Method GET -Path '/api/admin/roles/permissions' -Token $token
if (-not ($catalog.domains | Where-Object { $_.key -eq $Domain })) {
  throw "'$Domain' is not a permission area on this server (is the deploy live?)."
}
$roles = @(Invoke-Api -Method GET -Path '/api/admin/roles' -Token $token)
if ($roles.Count -eq 0) {
  Write-Host 'There are no custom roles. Super admins already see everything, so there is nothing to grant.' -ForegroundColor Yellow
  return
}

$view = "$Domain.view"
$manage = "$Domain.manage"
Write-Host ''
Write-Host "Custom roles (current $Domain access):"
for ($i = 0; $i -lt $roles.Count; $i++) {
  $r = $roles[$i]
  $perms = @($r.permissions)
  $has = 'none'
  if ($perms -contains $manage) { $has = 'manage' } elseif ($perms -contains $view) { $has = 'view' }
  Write-Host ("  [{0}] {1,-28} {2,-7} holds: {3}" -f ($i + 1), $r.name, $has, (($perms | Sort-Object) -join ', '))
}

# ---- choose -------------------------------------------------------------------------------------
Write-Host ''
$pick = Read-Host "Roles to grant $Domain to (numbers like 1,3 or 'all'; Enter to cancel)"
if (-not $pick) { Write-Host 'Cancelled.'; return }
if ($pick.Trim().ToLower() -eq 'all') {
  $chosen = $roles
} else {
  $chosen = @()
  foreach ($n in ($pick -split '[,\s]+' | Where-Object { $_ })) {
    $idx = 0
    if (-not [int]::TryParse($n, [ref]$idx) -or $idx -lt 1 -or $idx -gt $roles.Count) { throw "Not a role number: $n" }
    $chosen += $roles[$idx - 1]
  }
}
$level = (Read-Host "Level: 'view' (read only) or 'manage' (reply, notes, status, assignee) [manage]").Trim().ToLower()
if (-not $level) { $level = 'manage' }
if ($level -ne 'view' -and $level -ne 'manage') { throw "Level must be view or manage, not '$level'." }
$add = @($view)
if ($level -eq 'manage') { $add += $manage }

# ---- plan -----------------------------------------------------------------------------------------
$plan = @()
foreach ($r in $chosen) {
  $current = @($r.permissions)
  $missing = @($add | Where-Object { $current -notcontains $_ })
  if ($missing.Count -eq 0) {
    Write-Host "  = $($r.name): already has $($add -join ' + ')" -ForegroundColor DarkGray
    continue
  }
  $plan += [pscustomobject]@{ Role = $r; New = @($current + $missing); Added = $missing }
  Write-Host "  + $($r.name): add $($missing -join ', ')" -ForegroundColor Cyan
}
if ($plan.Count -eq 0) { Write-Host 'Nothing to change.'; return }
if ($DryRun) { Write-Host 'Dry run: no changes made.' -ForegroundColor Yellow; return }

$confirm = Read-Host "Apply these $($plan.Count) change(s) on $ApiBase ? Type 'yes'"
if ($confirm -ne 'yes') { Write-Host 'Cancelled, nothing changed.'; return }

# ---- apply ----------------------------------------------------------------------------------------
foreach ($p in $plan) {
  $r = $p.Role
  $body = @{ name = $r.name; description = "$($r.description)"; permissions = @($p.New) }
  $saved = Invoke-Api -Method PUT -Path "/api/admin/roles/$($r.id)" -Body $body -Token $token
  Write-Host "  OK $($saved.name): now holds $((@($saved.permissions) | Sort-Object) -join ', ')" -ForegroundColor Green
}
Write-Host ''
Write-Host 'Done. Holders of these roles get the new access on their next token refresh or login (signing out and back in is quickest).'

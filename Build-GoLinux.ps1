param(
  [ValidateSet("linux")]
  [string]$GOOS = "linux",

  [ValidateSet("amd64","arm64","arm","386")]
  [string]$GOARCH = "amd64",

  # Укажите для ARM (например "7"), иначе не используется
  [string]$GOARM,

  # Включить cgo? По умолчанию выключено (удобно для кросс-компиляции)
  [switch]$Cgo,

  # Пакет к main (используйте слэши вперед!)
  [string]$Package = "./chat-bot/cmd/chat-bot",

  # Путь к выходному бинарю
  [string]$Out = "dist/chat-bot-app",

  # Доп. флаги линкера (уменьшают размер)
  [string]$Ldflags = "-s -w",

  # Добавлять -trimpath?
  [switch]$Trimpath
)

# --- сохранить исходное окружение ---
$prev = @{
  GOOS        = $env:GOOS
  GOARCH      = $env:GOARCH
  GOARM       = $env:GOARM
  CGO_ENABLED = $env:CGO_ENABLED
}

function Restore-Env {
  foreach ($k in $prev.Keys) {
    if ($null -eq $prev[$k] -or $prev[$k] -eq "") {
      Remove-Item "Env:\$k" -ErrorAction SilentlyContinue
    } else {
      Set-Item "Env:\$k" $prev[$k]
    }
  }
}

try {
  # --- установить окружение для кросс-билда ---
  $env:GOOS  = $GOOS
  $env:GOARCH = $GOARCH
  if ($GOARM) { $env:GOARM = $GOARM } else { Remove-Item Env:\GOARM -ErrorAction SilentlyContinue }
  $env:CGO_ENABLED = ($(if ($Cgo) { "1" } else { "0" }))

  # --- подготовка выходной папки ---
  $outDir = Split-Path -Parent $Out
  if ($outDir -and -not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

  # --- собрать команду go build ---
  $args = @("build")
  if ($Trimpath) { $args += "-trimpath" }
  if ($Ldflags)  { $args += "-ldflags=$Ldflags" }
  $args += @("-o", $Out, $Package)

  Write-Host "GOOS=$($env:GOOS) GOARCH=$($env:GOARCH) GOARM=$($env:GOARM) CGO_ENABLED=$($env:CGO_ENABLED)"
  Write-Host "go $($args -join ' ')"

  # --- запуск билда ---
  & go @args
  if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

  Write-Host "✅ Built: $Out"
}
catch {
  Write-Error $_
  exit 1
}
finally {
  # --- вернуть окружение как было (удалит если изначально не было) ---
  Restore-Env
  Write-Host "🔄 Env restored."
}

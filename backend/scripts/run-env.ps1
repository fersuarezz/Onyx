Get-Content "$PSScriptRoot\..\.env" | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
    $key, $value = $_ -split '=', 2
    Set-Item -Path "Env:$($key.Trim())" -Value $value.Trim()
}

go run "$PSScriptRoot\..\main.go"

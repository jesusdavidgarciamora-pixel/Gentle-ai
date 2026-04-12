# Gentle AI Auto-Updater (Hybrid Sync)
# Mantiene tus cambios locales y los pone sobre lo último del original (Rebase)
# Recompila y sincroniza globalmente.

$GENTLE_AI_PATH = "D:\GitHub\gentle-ai"
$GENTLE_AI_BIN = "$GENTLE_AI_PATH\bin\gentle-ai.exe"

Write-Host "--- Iniciando Sincronización Híbrida (Tus Cambios + Original) ---" -ForegroundColor Cyan

# 1. Fetch de lo último del original
Write-Host "[1/4] Buscando novedades en Gentleman-Programming..." -ForegroundColor Yellow
git -C $GENTLE_AI_PATH fetch origin main

# 2. Rebase de tus cambios locales
Write-Host "[2/4] Integrando lo original con tus modificaciones (Rebase)..." -ForegroundColor Yellow
$rebaseResult = git -C $GENTLE_AI_PATH pull --rebase origin main 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "ATENCIÓN: Conflictos detectados." -ForegroundColor Red
    Write-Host $rebaseResult
    Write-Host "Hay cambios tuyos que chocan con los de Alan. Resolvé los conflictos en Git y terminá con 'git rebase --continue'." -ForegroundColor White
    exit 1
}

# 3. Recompilar el binario
Write-Host "[3/4] Recompilando gentle-ai.exe..." -ForegroundColor Yellow
go -C $GENTLE_AI_PATH build -o bin\gentle-ai.exe .\cmd\gentle-ai

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Falló la compilación de Go." -ForegroundColor Red
    exit 1
}

# 4. Sincronizar Globalmente
Write-Host "[4/4] Sincronizando todos tus agentes..." -ForegroundColor Yellow
& $GENTLE_AI_BIN sync

Write-Host "--- Gentle AI Actualizado con Éxito (Respetando tus Cambios) ---" -ForegroundColor Green

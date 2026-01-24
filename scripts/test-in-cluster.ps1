# AKS/K8s cluster'da optimizer ve ml-service pod'larının çalışıp çalışmadığını test eder.
# Kullanım: .\scripts\test-in-cluster.ps1 [namespace]
#   namespace: default: default

param(
    [string]$Namespace = "default"
)

$ErrorActionPreference = "Stop"

Write-Host "=============================================="
Write-Host "  k8s-resource-optimizer cluster test"
Write-Host "  namespace: $Namespace"
Write-Host "=============================================="

# 1) Pod durumları
Write-Host ""
Write-Host "--- 1) Pod durumları ---"
kubectl get pods -n $Namespace -l 'app in (k8s-resource-optimizer)' -o wide

# Running pod sayısını kontrol et
$optimizerPods = kubectl get pods -n $Namespace -l component=optimizer --no-headers 2>$null
$mlPods = kubectl get pods -n $Namespace -l component=ml-service --no-headers 2>$null

$runningOpt = 0
$runningMl = 0

if ($optimizerPods) {
    $runningOpt = ($optimizerPods | Select-String -Pattern "Running" -AllMatches).Matches.Count
}

if ($mlPods) {
    $runningMl = ($mlPods | Select-String -Pattern "Running" -AllMatches).Matches.Count
}

if ($runningOpt -lt 1 -or $runningMl -lt 1) {
    Write-Host ""
    Write-Host "Uyarı: Her iki pod da Running olmalı (optimizer: $runningOpt, ml-service: $runningMl)."
    exit 1
}

# 2) Son loglar (kısa)
Write-Host ""
Write-Host "--- 2) Optimizer son loglar ---"
try {
    kubectl logs -n $Namespace -l component=optimizer --tail=5 2>$null
} catch {
    Write-Host "(log yok)"
}

Write-Host ""
Write-Host "--- 3) ML service son loglar ---"
try {
    kubectl logs -n $Namespace -l component=ml-service --tail=5 2>$null
} catch {
    Write-Host "(log yok)"
}

# 3) Port-forward + HTTP testleri
Write-Host ""
Write-Host "--- 4) HTTP health kontrolleri (port-forward) ---"

$optimizerJob = $null
$mlJob = $null

try {
    # Port-forward job'ları başlat
    $optimizerJob = Start-Job -ScriptBlock {
        param($ns)
        kubectl port-forward -n $ns svc/k8s-resource-optimizer 8080:8080 2>$null
    } -ArgumentList $Namespace

    $mlJob = Start-Job -ScriptBlock {
        param($ns)
        kubectl port-forward -n $ns svc/ml-service 5000:5000 2>$null
    } -ArgumentList $Namespace

    # Port-forward'ların başlaması için bekle
    Start-Sleep -Seconds 3

    # Optimizer health
    Write-Host -NoNewline "  Optimizer /api/v1/health: "
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/health" -UseBasicParsing -TimeoutSec 5
        $content = $response.Content.Substring(0, [Math]::Min(200, $response.Content.Length))
        Write-Host $content
        Write-Host " [OK]"
    } catch {
        Write-Host " [FAIL]"
    }

    # ML health
    Write-Host -NoNewline "  ML service /health: "
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:5000/health" -UseBasicParsing -TimeoutSec 5
        $content = $response.Content.Substring(0, [Math]::Min(200, $response.Content.Length))
        Write-Host $content
        Write-Host " [OK]"
    } catch {
        Write-Host " [FAIL]"
    }

    # Optimizer recommendations (boş olabilir, 200 dönmeli)
    Write-Host -NoNewline "  Optimizer /api/v1/recommendations: "
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/recommendations" -TimeoutSec 5
        Write-Host " [OK]"
        
        if ($response.count -gt 0) {
            Write-Host "    → Toplam $($response.count) recommendation bulundu"
            Write-Host "    → İlk 3 pod:"
            $response.recommendations | Select-Object -First 3 | ForEach-Object {
                Write-Host "      - $($_.namespace)/$($_.pod_name)"
            }
        } else {
            Write-Host "    → Henüz recommendation yok (metrikler toplanıyor olabilir)"
        }
        Write-Host ""
    } catch {
        Write-Host " [FAIL]"
    }

    # ML /predict/all (minimal payload)
    Write-Host -NoNewline "  ML service /predict/all: "
    $payload = @{
        pod_name = "test"
        namespace = "default"
        metrics = @{
            c1 = @{
                cpu = @(
                    @{
                        timestamp = "2026-01-01T00:00:00Z"
                        value = 0.1
                    }
                )
                memory = @(
                    @{
                        timestamp = "2026-01-01T00:00:00Z"
                        value = 256
                    }
                )
            }
        }
    } | ConvertTo-Json -Depth 10

    try {
        $response = Invoke-RestMethod -Uri "http://localhost:5000/predict/all" `
            -Method Post `
            -ContentType "application/json" `
            -Body $payload `
            -TimeoutSec 5
        
        if ($response.success -eq $true) {
            Write-Host " [OK]"
        } else {
            Write-Host " [FAIL]"
        }
    } catch {
        Write-Host " [FAIL] (curl ile dene: curl -s -X POST http://localhost:5000/predict/all -H 'Content-Type: application/json' -d '$payload')"
    }

} finally {
    # Cleanup: Port-forward job'ları durdur
    if ($optimizerJob) {
        Stop-Job -Job $optimizerJob -ErrorAction SilentlyContinue
        Remove-Job -Job $optimizerJob -ErrorAction SilentlyContinue
    }
    if ($mlJob) {
        Stop-Job -Job $mlJob -ErrorAction SilentlyContinue
        Remove-Job -Job $mlJob -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Host "=============================================="
Write-Host "  Test tamamlandı."
Write-Host "=============================================="

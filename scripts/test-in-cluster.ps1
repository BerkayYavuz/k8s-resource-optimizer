# AKS/K8s cluster'da optimizer ve ml-service pod'larinin calisip calismadigini test eder.
# Kullanim: .\scripts\test-in-cluster.ps1 [namespace]
#   namespace: default: default

param(
    [string]$Namespace = "default"
)

$ErrorActionPreference = "Stop"

Write-Host "=============================================="
Write-Host "  k8s-resource-optimizer cluster test"
Write-Host "  namespace: $Namespace"
Write-Host "=============================================="

# 1) Pod durumlari
Write-Host ""
Write-Host "--- 1) Pod durumlari ---"
kubectl get pods -n $Namespace -l "app=k8s-resource-optimizer" -o wide

$optOut = kubectl get pods -n $Namespace -l component=optimizer --no-headers 2>$null
$mlOut  = kubectl get pods -n $Namespace -l component=ml-service --no-headers 2>$null
$runningOpt = 0
$runningMl  = 0
if ($optOut) { $runningOpt = @($optOut | Where-Object { $_ -match "Running" }).Count }
if ($mlOut)  { $runningMl  = @($mlOut  | Where-Object { $_ -match "Running" }).Count }

if ($runningOpt -lt 1 -or $runningMl -lt 1) {
    Write-Host ""
    Write-Host "Uyari: Her iki pod da Running olmali (optimizer: $runningOpt, ml-service: $runningMl)."
    exit 1
}

# 2) Son loglar (kisa)
Write-Host ""
Write-Host "--- 2) Optimizer son loglar ---"
try {
    kubectl logs -n $Namespace -l component=optimizer --tail=5 2>$null
} catch { Write-Host "(log yok)" }

Write-Host ""
Write-Host "--- 3) ML service son loglar ---"
try {
    kubectl logs -n $Namespace -l component=ml-service --tail=5 2>$null
} catch { Write-Host "(log yok)" }

# 4) Port-forward + HTTP testleri
Write-Host ""
Write-Host "--- 4) HTTP health kontrolleri (port-forward) ---"

$optJob = $null
$mlJob  = $null

try {
    $optJob = Start-Job -ScriptBlock { param($ns) kubectl port-forward -n $ns svc/k8s-resource-optimizer 8080:8080 2>$null } -ArgumentList $Namespace
    $mlJob  = Start-Job -ScriptBlock { param($ns) kubectl port-forward -n $ns svc/ml-service 5000:5000 2>$null } -ArgumentList $Namespace
    Start-Sleep -Seconds 3

    # Optimizer health
    Write-Host -NoNewline "  Optimizer /api/v1/health: "
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/health" -TimeoutSec 5
        Write-Host "[OK]"
        if ($r.status) { Write-Host "    -> status: $($r.status)" }
    } catch { Write-Host "[FAIL]" }

    # ML health
    Write-Host -NoNewline "  ML service /health: "
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:5000/health" -TimeoutSec 5
        Write-Host "[OK]"
        if ($r.status) { Write-Host "    -> status: $($r.status)" }
    } catch { Write-Host "[FAIL]" }

    # Optimizer recommendations (detayli)
    Write-Host -NoNewline "  Optimizer /api/v1/recommendations: "
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/recommendations" -TimeoutSec 10
        Write-Host "[OK]"
        $count = 0
        if ($null -ne $r.count) { $count = [int]$r.count }
        Write-Host "    -> Toplam: $count oneri"

        if ($count -gt 0 -and $r.recommendations) {
            $recs = @($r.recommendations)

            # Namespace dagilimi
            $nsGroups = $recs | Group-Object -Property namespace | Sort-Object { -$_.Count }
            $nsSummary = ($nsGroups | ForEach-Object { "$($_.Name): $($_.Count)" }) -join ", "
            Write-Host "    -> Namespace: $nsSummary"

            # Toplam tasarruf
            $totalCpu = 0
            $totalMem = 0
            foreach ($rec in $recs) {
                if ($null -ne $rec.potential_cpu_saving_cores) {
                    $totalCpu += [double]$rec.potential_cpu_saving_cores
                }
                if ($null -ne $rec.potential_memory_saving_mb) {
                    $totalMem += [long]$rec.potential_memory_saving_mb
                }
            }
            
            $tcStr = [string][Math]::Abs($totalCpu)
            $tmStr = [string][Math]::Abs($totalMem)
            if ($tcStr.Length -le 10 -and $tmStr.Length -le 12) {
                $cpuRounded = [Math]::Round($totalCpu, 3)
                Write-Host "    -> Tasarruf: CPU $cpuRounded core, bellek $totalMem MB"
            } else {
                Write-Host "    -> Tasarruf: (ozet atlandi - asiri degerler)"
            }

            # Oneri ozeti (en fazla 8)
            Write-Host "    -> Oneri ozeti (en fazla 8):"
            $first8 = $recs | Select-Object -First 8
            foreach ($rec in $first8) {
                $nCont = 0
                if ($rec.containers -and $rec.containers.PSObject.Properties) {
                    $nCont = @($rec.containers.PSObject.Properties).Count
                }
                
                $cpu = 0
                if ($null -ne $rec.potential_cpu_saving_cores) {
                    $cpu = [Math]::Round([double]$rec.potential_cpu_saving_cores, 3)
                }
                
                $mem = 0
                if ($null -ne $rec.potential_memory_saving_mb) {
                    $mem = [long]$rec.potential_memory_saving_mb
                }
                
                $conf = 0
                if ($null -ne $rec.overall_confidence) {
                    $conf = [Math]::Floor([double]$rec.overall_confidence * 100)
                }
                
                Write-Host "      $($rec.namespace)/$($rec.pod_name) | $nCont container | CPU $cpu core, Mem $mem MB | guven %$conf"
            }
            
            if ($count -gt 8) {
                $more = $count - 8
                Write-Host "      ... ve $more oneri daha"
            }
        }
    } catch {
        Write-Host "[FAIL]"
    }

    # ML /predict/all (minimal payload)
    Write-Host -NoNewline "  ML service /predict/all: "
    $payload = @'
{"pod_name":"test","namespace":"default","metrics":{"c1":{"cpu":[{"timestamp":"2026-01-01T00:00:00Z","value":0.1}],"memory":[{"timestamp":"2026-01-01T00:00:00Z","value":256}]}}}
'@
    
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:5000/predict/all" -Method Post -ContentType "application/json" -Body $payload -TimeoutSec 5
        if ($r.success -eq $true) {
            Write-Host "[OK]"
        } else {
            Write-Host "[FAIL]"
        }
    } catch {
        Write-Host "[FAIL]"
    }

} finally {
    if ($optJob) {
        Stop-Job -Job $optJob -ErrorAction SilentlyContinue
        Remove-Job -Job $optJob -ErrorAction SilentlyContinue
    }
    if ($mlJob) {
        Stop-Job -Job $mlJob -ErrorAction SilentlyContinue
        Remove-Job -Job $mlJob -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Host "=============================================="
Write-Host "  Test tamamlandi."
Write-Host "=============================================="

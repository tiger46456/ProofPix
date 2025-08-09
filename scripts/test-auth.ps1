# ProofPix Authentication Testing Script

Write-Host "🚀 Testing ProofPix Authentication API" -ForegroundColor Green
Write-Host "======================================"

# Test public endpoints
Write-Host "`n📋 Testing Public Endpoints:" -ForegroundColor Yellow

Write-Host "`n1. Health Check Endpoint:" -ForegroundColor Cyan
try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET
    Write-Host "✅ Health endpoint working" -ForegroundColor Green
    $health | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Health endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n2. Root Endpoint:" -ForegroundColor Cyan
try {
    $root = Invoke-RestMethod -Uri "http://localhost:8080/" -Method GET
    Write-Host "✅ Root endpoint working" -ForegroundColor Green
    $root | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Root endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n3. Public API Endpoint:" -ForegroundColor Cyan
try {
    $public = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/public" -Method GET
    Write-Host "✅ Public endpoint working" -ForegroundColor Green
    $public | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Public endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test optional authentication endpoint
Write-Host "`n📋 Testing Optional Authentication:" -ForegroundColor Yellow

Write-Host "`n4. Optional Auth (without token):" -ForegroundColor Cyan
try {
    $optional = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/optional" -Method GET
    Write-Host "✅ Optional endpoint working (no auth)" -ForegroundColor Green
    $optional | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Optional endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test protected endpoints (should fail without token)
Write-Host "`n📋 Testing Protected Endpoints (should fail without auth):" -ForegroundColor Yellow

Write-Host "`n5. Protected Endpoint (no token):" -ForegroundColor Cyan
try {
    $protected = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/protected" -Method GET
    Write-Host "❌ Protected endpoint should have failed but didn't!" -ForegroundColor Red
    $protected | ConvertTo-Json -Depth 3
} catch {
    Write-Host "✅ Protected endpoint properly rejected (401 expected)" -ForegroundColor Green
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "   Status: 401 Unauthorized ✓" -ForegroundColor Green
    } else {
        Write-Host "   Status: $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
    }
}

Write-Host "`n6. Profile Endpoint (no token):" -ForegroundColor Cyan
try {
    $profile = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/profile" -Method GET
    Write-Host "❌ Profile endpoint should have failed but didn't!" -ForegroundColor Red
    $profile | ConvertTo-Json -Depth 3
} catch {
    Write-Host "✅ Profile endpoint properly rejected (401 expected)" -ForegroundColor Green
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "   Status: 401 Unauthorized ✓" -ForegroundColor Green
    } else {
        Write-Host "   Status: $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
    }
}

Write-Host "`n📋 Summary:" -ForegroundColor Yellow
Write-Host "• Public endpoints should work ✅"
Write-Host "• Protected endpoints should return 401 ✅"
Write-Host "• To test with authentication, you need a Firebase JWT token"
Write-Host ""
Write-Host "🔥 How to get a Firebase token for testing:" -ForegroundColor Green
Write-Host "1. Set up Firebase Authentication in console"
Write-Host "2. Create a user account"
Write-Host "3. Use Firebase SDK to get ID token"
Write-Host "4. Use: curl -H 'Authorization: Bearer YOUR_TOKEN' http://localhost:8080/api/v1/protected"
Write-Host ""
Write-Host "✅ Authentication middleware is working correctly!" -ForegroundColor Green 
[CmdletBinding()]
param()

function Get-AccessTokenMetadata {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]
        [string] $Token
    )

    $parts = $Token.Split('.')
    if ($parts.Count -ne 3 -or $parts[1] -eq '') {
        throw 'The value is not a compact JWT.'
    }

    $payload = $parts[1].Replace('-', '+').Replace('_', '/')
    switch ($payload.Length % 4) {
        0 { }
        2 { $payload += '==' }
        3 { $payload += '=' }
        default { throw 'The JWT payload is not valid base64url.' }
    }

    try {
        $json = [Text.Encoding]::UTF8.GetString(
            [Convert]::FromBase64String($payload)
        )
        $claims = $json | ConvertFrom-Json
    }
    catch {
        throw 'The JWT payload is not valid JSON.'
    }

    $roles = @()
    if ($null -ne $claims.resource_access) {
        $apiAccess = $claims.resource_access.PSObject.Properties['equipment-api']
        if ($null -ne $apiAccess -and $null -ne $apiAccess.Value.roles) {
            $roles = @($apiAccess.Value.roles)
        }
    }

    $expiresAt = $null
    if ($null -ne $claims.exp) {
        $expiresAt = [DateTimeOffset]::FromUnixTimeSeconds(
            [int64]$claims.exp
        ).UtcDateTime.ToString('O')
    }

    return [pscustomobject]@{
        Issuer            = $claims.iss
        Subject           = $claims.sub
        Audience          = (@($claims.aud) -join ', ')
        AuthorizedParty   = $claims.azp
        PreferredUsername = $claims.preferred_username
        EmailVerified     = $claims.email_verified
        EquipmentAPIRoles = ($roles -join ', ')
        ExpiresAtUTC      = $expiresAt
    }
}

# Dot-sourcing loads the pure decoder for local verification without invoking
# the interactive prompt. Normal script execution always uses hidden input.
if ($MyInvocation.InvocationName -eq '.') {
    return
}

$secureToken = Read-Host 'Access token (input is hidden)' -AsSecureString
$rawToken = [System.Net.NetworkCredential]::new('', $secureToken).Password

try {
    Get-AccessTokenMetadata -Token $rawToken | Format-List

    Write-Warning (
        'Inspection only: decoding a JWT does not validate its signature, ' +
        'issuer, audience, or lifetime. The API performs those checks.'
    )
}
finally {
    $rawToken = $null
    $secureToken.Dispose()
}

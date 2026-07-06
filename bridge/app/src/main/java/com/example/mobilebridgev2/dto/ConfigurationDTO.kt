package com.example.mobilebridgev2.dto

data class ConfigurationDTO (
    var deviceId: Int,
    var touchBalanceCheckUssd: String,
    var alfaBalanceCheckUssd: String,
    var touchThirdPartyRechargeTemplate: String,
    var touchCreditTransferSmsTemplate: String,
    var touchCreditTransferDestination: String,
    var alfaThirdPartyRechargeSmsTemplate: String,
    var alfaCreditTransferDestination: String,
    var alfaCreditTransferSmsTemplate: String,
    var alfaThirdPartyRechargeDestination: String,
    var touchMinimumAllowedBalance: Double,
    var alfaMinimumAllowedBalance: Double,
    var touchSimBalance: Double,
    var touchSimValidityDate: String,
    var touchCreditTransferMessageFee: Double,
    var alfaCreditTransferMessageFee: Double,
    // Additive fields (defaulted so an older server that omits them still decodes):
    // alfa balance/validity echo (restores state after reinstall) + control knobs
    // + server-configurable reply matching.
    var alfaSimBalance: Double = 0.0,
    var alfaSimValidityDate: String = "",
    var pollIntervalSeconds: Int = 5,
    var heartbeatIntervalSeconds: Int = 300,
    var maxSmsPerHalfHour: Int = 25,
    var successMatchPatterns: List<String> = emptyList(),
    var failureMatchPatterns: List<String> = emptyList()
)

package com.example.mobilebridgev2.dto

data class ConfigurationDTO (
    // Mongo ObjectID hex string (server sends d.ID.Hex()), NOT a number. Typing it
    // as Int made Gson throw NumberFormatException on the whole config payload, so
    // the device silently fell back to hardcoded defaults (e.g. the 20.0 min-balance
    // reserve) and never received the server's real config.
    var deviceId: String,
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

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
    var alfaCreditTransferMessageFee: Double
)

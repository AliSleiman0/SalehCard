package com.example.mobilebridgev2

import android.telephony.SubscriptionInfo
import java.time.LocalDate

object ProviderStore {
    var deviceId: Int = 0
    lateinit var touchSim: SubscriptionInfo
    lateinit var alfaSim: SubscriptionInfo
    var touchBalanceCheckUssd: String = "*220#"
    var alfaBalanceCheckUssd: String = "*11#"
    var touchThirdPartyRechargeTemplate: String = "*300*{phone}#{card}"
    var touchCreditTransferSmsTemplate: String = "{phone}T{amount}"
    var touchCreditTransferDestination: String = "1199"
    var alfaThirdPartyRechargeSmsTemplate: String = "{phone}R{code}"
    var alfaCreditTransferDestination: String = "1399"
    var alfaCreditTransferSmsTemplate: String = "{phone}T{amount}"
    var alfaThirdPartyRechargeDestination: String = "1313"
    var touchMinimumAllowedBalance: Double = 20.00
    var alfaMinimumAllowedBalance: Double = 20.00
    var touchSimBalance: Double = 0.00
    var touchCreditTransferMessageFee: Double = 0.16
    var alfaCreditTransferMessageFee: Double = 0.14
    var touchSimValidityDate: String = LocalDate.now().toString()
}
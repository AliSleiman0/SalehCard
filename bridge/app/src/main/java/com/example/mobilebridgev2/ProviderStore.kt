package com.example.mobilebridgev2

import android.telephony.SubscriptionInfo
import java.time.LocalDate

object ProviderStore {
    var deviceId: String = ""
    // Nullable (not lateinit): a phone may carry only one operator's SIM, so a
    // command for the missing operator must fail cleanly instead of crashing on
    // an uninitialized lateinit. Use hasSim() to guard before touching these.
    var touchSim: SubscriptionInfo? = null
    var alfaSim: SubscriptionInfo? = null

    // hasSim reports whether the SIM for a provider was detected at startup.
    fun hasSim(provider: String): Boolean = when (provider.lowercase()) {
        "touch" -> touchSim != null
        "alfa" -> alfaSim != null
        else -> false
    }
    var touchBalanceCheckUssd: String = "*220#"
    var alfaBalanceCheckUssd: String = "*11#"
    var touchThirdPartyRechargeTemplate: String = "*300*961{phone}*{card}#"
    var touchCreditTransferSmsTemplate: String = "{phone}T{amount}"
    var touchCreditTransferDestination: String = "1199"
    // Alfa recharge is a USSD dial (*111*{code}*{phone}#), not SMS despite the
    // legacy field name. Overridden by server config at startup.
    var alfaThirdPartyRechargeSmsTemplate: String = "*111*{code}*{phone}#"
    var alfaCreditTransferDestination: String = "1313"
    var alfaCreditTransferSmsTemplate: String = "{phone}T{amount}"
    var alfaThirdPartyRechargeDestination: String = "1313"
    var touchMinimumAllowedBalance: Double = 20.00
    var alfaMinimumAllowedBalance: Double = 20.00
    var touchSimBalance: Double = 0.00
    var touchCreditTransferMessageFee: Double = 0.16
    var alfaCreditTransferMessageFee: Double = 0.14
    var touchSimValidityDate: String = LocalDate.now().toString()
    // Alfa balance/validity are tracked separately from touch (the old build only
    // ever wrote the touch fields — see checkBalance).
    var alfaSimBalance: Double = 0.00
    var alfaSimValidityDate: String = LocalDate.now().toString()
    // Control knobs + reply matching, pushed from the server config.
    var pollIntervalSeconds: Int = 5
    var heartbeatIntervalSeconds: Int = 300
    var maxSmsPerHalfHour: Int = 25
    var successMatchPatterns: List<String> = listOf("success", "transferred")
    var failureMatchPatterns: List<String> = listOf("fail", "do not have", "insufficient")
}
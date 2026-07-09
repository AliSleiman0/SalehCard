package com.example.mobilebridgev2

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.Build
import android.telephony.SmsManager
import android.telephony.TelephonyManager
import androidx.annotation.RequiresPermission
import com.example.mobilebridgev2.dto.CommandDTO
import com.example.mobilebridgev2.dto.CommandResultCodes
import com.example.mobilebridgev2.dto.CommandResultDTO
import com.example.mobilebridgev2.sms.SmsReplyRouter
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import kotlin.math.roundToInt
import android.os.Handler
import android.os.Looper
import androidx.annotation.RequiresApi
import com.example.mobilebridgev2.net.BridgeReporter
import com.example.mobilebridgev2.ussd.AccessibilityUtil
import com.example.mobilebridgev2.ussd.AlfaUssdAccessibilityService
import com.example.mobilebridgev2.ussd.UssdSessionActivity
import com.example.mobilebridgev2.ussd.UssdSessionCoordinator
import android.os.PowerManager
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.withTimeoutOrNull
import java.util.UUID

object CommandExecutor {

    fun getBalanceFromReply(reply: String): Pair<Double, String>{
        // Touch and Alfa answer their balance USSD in two different shapes; each is
        // matched by (amount, validity) capture group so the balance/date come
        // straight off the match — no positional split/substring, which only ever
        // fit the touch layout and left every alfa balance parsed as 0.00.
        //   touch (*220#): "USD 5.00 Exp:10-06-27"    — currency-first, "Exp:", dash date, 2- or 4-digit year
        //   alfa  (*11#):  "1.01 USD till 04/09/2026" — amount-first, "till", slash date
        // The year width stays 2–4 digits so a future format tweak doesn't regress.
        val patterns = listOf(
            Regex("""USD\s+(\d+(?:\.\d{1,2})?)\s+Exp:\s*(\d{2}[-/]\d{2}[-/]\d{2,4})""", RegexOption.IGNORE_CASE),
            Regex("""(\d+(?:\.\d{1,2})?)\s*USD\s+till\s+(\d{2}[-/]\d{2}[-/]\d{2,4})""", RegexOption.IGNORE_CASE),
        )
        for (pattern in patterns) {
            val match = pattern.find(reply) ?: continue
            val balance = match.groupValues[1].toDoubleOrNull() ?: continue
            return Pair(balance, match.groupValues[2])
        }
        return Pair(0.0, "0")
    }

    fun splitAmount(amount: Int, maxChunk: Int = 3): List<Int> {
        val chunks = mutableListOf<Int>()
        var remaining = amount

        while (remaining > 0) {
            val chunk = minOf(remaining, maxChunk)
            chunks.add(chunk)
            remaining -= chunk
        }

        return chunks
    }
    @RequiresApi(Build.VERSION_CODES.S)
    @Suppress("DEPRECATION")
    suspend fun sendSms(
        context: Context,
        provider: String,
        destination: String,
        message: String
    ): Boolean {
        return suspendCancellableCoroutine { continuation ->

            val action = "SMS_SENT_${UUID.randomUUID()}"

            val sentIntent = Intent(action)

            val pendingIntent = PendingIntent.getBroadcast(
                context,
                action.hashCode(),
                sentIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )

            val receiver = object : BroadcastReceiver() {
                override fun onReceive(context: Context?, intent: Intent?) {

                    try {
                        context?.unregisterReceiver(this)
                    } catch (_: Exception) {
                    }

                    val success = resultCode == Activity.RESULT_OK

                    if (continuation.isActive) {
                        continuation.resume(success)
                    }
                }
            }

            try {
                val filter = IntentFilter(action)

                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    // The SMS "sent" status is delivered by the telephony system
                    // process through this PendingIntent's broadcast. A
                    // RECEIVER_NOT_EXPORTED context receiver never fires for that
                    // cross-process broadcast on Android 13+, so sendSms would time
                    // out and report a send failure even though the radio actually
                    // transmitted the SMS (the "transfer succeeded but shows failed"
                    // bug). The action carries a random per-send UUID, so an exported
                    // receiver can't be spoofed by another app.
                    context.registerReceiver(
                        receiver,
                        filter,
                        Context.RECEIVER_EXPORTED
                    )
                } else {
                    context.registerReceiver(receiver, filter)
                }

                val subscriptionId =
                    if (provider.lowercase() == "touch") {
                        ProviderStore.touchSim!!.subscriptionId
                    } else {
                        ProviderStore.alfaSim!!.subscriptionId
                    }

                val defaultSmsManager = context.getSystemService(SmsManager::class.java)
                val smsManager = defaultSmsManager.createForSubscriptionId(subscriptionId)

                smsManager.sendTextMessage(
                    destination,
                    null,
                    message,
                    pendingIntent,
                    null
                )

                continuation.invokeOnCancellation {
                    try {
                        context.unregisterReceiver(receiver)
                    } catch (_: Exception) {
                    }
                }

            } catch (e: Exception) {
                try {
                    context.unregisterReceiver(receiver)
                } catch (_: Exception) {
                }

                e.printStackTrace()

                if (continuation.isActive) {
                    continuation.resume(false)
                }
            }
        }
    }

    @SuppressLint("MissingPermission")
    suspend fun sendUssd(
        context: Context,
        subscriptionId: Int?,
        ussdCode: String
    ): String? {
        if (subscriptionId == null) return null

        return suspendCancellableCoroutine { continuation ->

            val telephonyManager =
                context.getSystemService(TelephonyManager::class.java)

            val simTelephonyManager =
                telephonyManager.createForSubscriptionId(subscriptionId)

            simTelephonyManager.sendUssdRequest(
                ussdCode,
                object : TelephonyManager.UssdResponseCallback() {

                    override fun onReceiveUssdResponse(
                        telephonyManager: TelephonyManager?,
                        request: String?,
                        response: CharSequence?
                    ) {
                        if (continuation.isActive) {
                            continuation.resume(response?.toString())
                        }
                    }

                    override fun onReceiveUssdResponseFailed(
                        telephonyManager: TelephonyManager?,
                        request: String?,
                        failureCode: Int
                    ) {
                        if (continuation.isActive) {
                            continuation.resume(null)
                        }
                    }
                },
                Handler(Looper.getMainLooper())
            )
        }
    }

    @RequiresApi(Build.VERSION_CODES.S)
    suspend fun smsTransfer(command: CommandDTO, context: Context): CommandResultDTO{
        if(command.recipientNumber == null){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.SMS_NO_RECIPIENT,
                null,
                null,
                0.0,
                command.provider,
                null,
                null,
                "SMS request without specific recipient"
            )
        }
        if(command.message == null){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.SMS_NO_MESSAGE,
                null,
                null,
                0.0,
                command.provider,
                null,
                null,
                "SMS request with empty message"
            )
        }
        val sentSuccessfully = withTimeoutOrNull(30_000L) {
            sendSms(
                context = context,
                provider = command.provider.lowercase(),
                destination = command.recipientNumber,
                message = command.message
            )
        } ?: false
        if(!sentSuccessfully){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.SMS_SEND_FAILED,
                null,
                null,
                0.0,
                command.provider,
                null,
                null,
                "SMS request incomplete, unable to send SMS"
            )
        }

        return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.SMS_SUCCESS,
            null,
            null,
            0.02,
            command.provider,
            0.0,
            null,
            null
        )
    }

    @RequiresApi(Build.VERSION_CODES.S)
    suspend fun transferCredits(
        command: CommandDTO,
        context: Context
    ): CommandResultDTO {

        if (command.recipientNumber == null) {
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.CREDIT_TRANSFER_NO_RECIPIENT,
                null,
                null,
                0.0,
                command.provider,
                null,
                null,
                "Credit transfer request has no recipient number"
            )
        }

        if (command.amount == null) {
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.CREDIT_TRANSFER_NO_AMOUNT,
                null,
                null,
                0.0,
                command.provider,
                null,
                null,
                "Credit transfer request has no specified amount"
            )
        }

        val isTouch = command.provider.equals(
            other = "touch",
            ignoreCase = true
        )

        val chunks = splitAmount(command.amount.roundToInt())

        val totalChunks = chunks.size

        val smsTemplate =
            if (isTouch) {
                ProviderStore.touchCreditTransferSmsTemplate
            } else {
                ProviderStore.alfaCreditTransferSmsTemplate
            }

        val destination =
            if (isTouch) {
                ProviderStore.touchCreditTransferDestination
            } else {
                ProviderStore.alfaCreditTransferDestination
            }

        val messageFee =
            if (isTouch) {
                ProviderStore.touchCreditTransferMessageFee
            } else {
                ProviderStore.alfaCreditTransferMessageFee
            }

        val requiredBalance =
            if (isTouch) {
                command.amount +
                        (totalChunks * ProviderStore.touchCreditTransferMessageFee) +
                        ProviderStore.touchMinimumAllowedBalance
            } else {
                0.0
            }

        if (
            isTouch &&
            ProviderStore.touchSimBalance < requiredBalance
        ) {
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.CREDIT_TRANSFER_INSUFFICIENT_BALANCE,
                0.0,
                null,
                0.0,
                command.provider,
                ProviderStore.touchSimBalance,
                ProviderStore.touchSimValidityDate,
                "Not enough Touch prepaid credit to complete the transfer"
            )
        }

        var billingAmount = 0.0
        var processedAmount = 0.0
        var processedChunks = 0

        for ((index, chunk) in chunks.withIndex()) {

            val currentChunkNumber = index + 1

            val message = smsTemplate
                .replace("{phone}", command.recipientNumber)
                .replace("{amount}", chunk.toString())

            val replyWaiter = SmsReplyRouter.prepareReplyWait(
                listOf(
                    command.provider,
                    destination
                )
            )

            val sentSuccessfully = withTimeoutOrNull(30_000L) {
                sendSms(
                    context = context,
                    provider = command.provider.lowercase(),
                    destination = destination,
                    message = message
                )
            } ?: false

            if (!sentSuccessfully) {
                // The system "sent" ack is unreliable — it can be dropped even when
                // the SMS was actually transmitted — so a missing ack is NOT a hard
                // failure. The operator's reply below is the source of truth: a
                // genuine send failure yields no reply and falls through to the
                // reply-timeout branch, while a merely-lost ack still receives the
                // "transferred" confirmation and completes. This stops transfers that
                // really went through from being flagged failed on the Bridge page.
                BridgeReporter.log(
                    "WARN",
                    "Transfer chunk $currentChunkNumber/$totalChunks: SMS sent-ack " +
                            "missing; awaiting operator reply as confirmation"
                )
            }

            val reply = withTimeoutOrNull(30_000L) {
                replyWaiter.await()
            }

            if (reply == null) {
                SmsReplyRouter.clearReplyWait(replyWaiter)

                val statusCode =
                    if (processedChunks > 0) {
                        CommandResultCodes.CREDIT_TRANSFER_PARTIALLY_COMPLETED
                    } else {
                        CommandResultCodes.CREDIT_TRANSFER_REPLY_TIMEOUT
                    }

                return CommandResultDTO(
                    command.commandId,
                    command.recipientNumber,
                    command.commandType,
                    System.currentTimeMillis(),
                    statusCode,
                    processedAmount,
                    null,
                    billingAmount,
                    command.provider,
                    if (isTouch) ProviderStore.touchSimBalance else null,
                    if (isTouch) ProviderStore.touchSimValidityDate else null,
                    if (processedChunks > 0) {
                        "Credit transfer partially completed. " +
                                "$processedChunks/$totalChunks chunks succeeded, " +
                                "transferring $processedAmount. " +
                                "Timed out waiting for the reply to chunk " +
                                "$currentChunkNumber/$totalChunks."
                    } else {
                        "Timed out waiting for provider reply on chunk " +
                                "$currentChunkNumber/$totalChunks."
                    }
                )
            }

            val isSuccess =
                reply.body.contains(
                    "success",
                    ignoreCase = true
                ) ||
                        reply.body.contains(
                            "transferred",
                            ignoreCase = true
                        )

            val isOutOfBalance =
                reply.body.contains(
                    "do not have",
                    ignoreCase = true
                )

            if (isOutOfBalance) {
                SmsReplyRouter.clearReplyWait(replyWaiter)

                if (isTouch) {
                    // The SIM ran out of prepaid credit mid-transfer. The admin sees
                    // this on the Bridge page (failed command + balance) and tops up
                    // the SIM; the order stays in the manual queue.
                    BridgeReporter.log("WARN", "Touch SIM out of balance during transfer")
                }

                val statusCode =
                    if (processedChunks > 0) {
                        CommandResultCodes.CREDIT_TRANSFER_PARTIALLY_COMPLETED
                    } else {
                        CommandResultCodes.CREDIT_TRANSFER_INSUFFICIENT_BALANCE
                    }

                return CommandResultDTO(
                    command.commandId,
                    command.recipientNumber,
                    command.commandType,
                    System.currentTimeMillis(),
                    statusCode,
                    processedAmount,
                    null,
                    billingAmount,
                    command.provider,
                    if (isTouch) ProviderStore.touchSimBalance else null,
                    if (isTouch) ProviderStore.touchSimValidityDate else null,
                    if (processedChunks > 0) {
                        "Credit transfer partially completed. " +
                                "$processedChunks/$totalChunks chunks succeeded, " +
                                "transferring $processedAmount. " +
                                "The provider reported insufficient balance " +
                                "on chunk $currentChunkNumber/$totalChunks."
                    } else {
                        "The provider reported insufficient balance."
                    }
                )
            }

            if (!isSuccess) {
                SmsReplyRouter.clearReplyWait(replyWaiter)

                val statusCode =
                    if (processedChunks > 0) {
                        CommandResultCodes.CREDIT_TRANSFER_PARTIALLY_COMPLETED
                    } else {
                        CommandResultCodes.CREDIT_TRANSFER_PROVIDER_REJECTED
                    }

                return CommandResultDTO(
                    command.commandId,
                    command.recipientNumber,
                    command.commandType,
                    System.currentTimeMillis(),
                    statusCode,
                    processedAmount,
                    null,
                    billingAmount,
                    command.provider,
                    if (isTouch) ProviderStore.touchSimBalance else null,
                    if (isTouch) ProviderStore.touchSimValidityDate else null,
                    if (processedChunks > 0) {
                        "Credit transfer partially completed. " +
                                "$processedChunks/$totalChunks chunks succeeded, " +
                                "transferring $processedAmount. " +
                                "The provider rejected chunk " +
                                "$currentChunkNumber/$totalChunks: ${reply.body}"
                    } else {
                        "Credit transfer rejected on chunk " +
                                "$currentChunkNumber/$totalChunks: ${reply.body}"
                    }
                )
            }

            /*
             * Only update financial state after the provider has confirmed
             * that this specific chunk succeeded.
             */
            val currentBillingAmount =
                chunk.toDouble() + messageFee

            processedAmount += chunk.toDouble()
            processedChunks++
            billingAmount += currentBillingAmount

            if (isTouch) {
                ProviderStore.touchSimBalance -= currentBillingAmount
            }
        }

        return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.CREDIT_TRANSFER_SUCCESS,
            processedAmount,
            null,
            billingAmount,
            command.provider,
            if (isTouch) ProviderStore.touchSimBalance else null,
            if (isTouch) ProviderStore.touchSimValidityDate else null,
            null
        )
    }

    @RequiresPermission(Manifest.permission.CALL_PHONE)
    suspend fun checkBalance(
        command: CommandDTO,
        context: Context): CommandResultDTO {

        val subscriptionId = if(command.provider == "touch") ProviderStore.touchSim!!.subscriptionId
        else ProviderStore.alfaSim!!.subscriptionId

        val ussdCode = if(command.provider == "touch") ProviderStore.touchBalanceCheckUssd
        else ProviderStore.alfaBalanceCheckUssd

        val reply = withTimeoutOrNull(30_000L) {
            sendUssd(
                context = context,
                subscriptionId = subscriptionId,
                ussdCode = ussdCode
            )
        } ?: return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.USSD_SEND_FAILED,
            command.amount,
            null,
            null,
            command.provider,
            null,
            null,
            "USSD check did not respond"
        )

        val (balance, validityDate) = getBalanceFromReply(reply)

        if(validityDate == "0") return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.USSD_REPLY_PARSE_FAILED,
            command.amount,
            null,
            null,
            command.provider,
            null,
            null,
            reply
        )

        // Write the balance/validity to the queried provider's slot — the old
        // build always wrote the touch fields even for an alfa balance check.
        if (command.provider.equals("touch", ignoreCase = true)) {
            ProviderStore.touchSimBalance = balance
            ProviderStore.touchSimValidityDate = validityDate
        } else {
            ProviderStore.alfaSimBalance = balance
            ProviderStore.alfaSimValidityDate = validityDate
        }

        return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.BALANCE_CHECK_SUCCESS,
            command.amount,
            null,
            null,
            command.provider,
            balance,
            validityDate,
            null
        )
        
    }

    suspend fun rechargeTouch(command: CommandDTO, context: Context): CommandResultDTO {
        if(command.recipientNumber == null){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.TOUCH_RECHARGE_NO_RECIPIENT,
                null,
                null,
                0.00,
                command.provider,
                null,
                null,
                "Recharge request with no recipient number"
            )
        }

        if(command.cardCode == null){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.TOUCH_RECHARGE_NO_CARD_CODE,
                null,
                null,
                0.02,
                command.provider,
                null,
                null,
                "credit transfer request with no card code"
            )
        }

        val ussdCode = ProviderStore.touchThirdPartyRechargeTemplate
            .replace("{phone}",command.recipientNumber)
            .replace("{card}",command.cardCode)

        val reply = withTimeoutOrNull(30_000L) {
            sendUssd(
                context = context,
                subscriptionId = ProviderStore.touchSim!!.subscriptionId,
                ussdCode = ussdCode
            )
        } ?: return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.TOUCH_RECHARGE_REPLY_TIMEOUT,
            command.amount,
            null,
            null,
            command.provider,
            null,
            null,
            "No USSD reply detected"
        )

        if(reply.lowercase().contains("fail")) return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.TOUCH_RECHARGE_PROVIDER_REJECTED,
            command.amount,
            null,
            null,
            command.provider,
            null,
            null,
            null
        )

        return CommandResultDTO(
            command.commandId,
            command.recipientNumber,
            command.commandType,
            System.currentTimeMillis(),
            CommandResultCodes.TOUCH_RECHARGE_SUCCESS,
            command.amount,
            null,
            null,
            command.provider,
            null,
            null,
            null
        )
    }
    @RequiresApi(Build.VERSION_CODES.S)
    suspend fun rechargeAlfa(command: CommandDTO, context: Context): CommandResultDTO {
        if(command.recipientNumber == null){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.ALFA_RECHARGE_NO_RECIPIENT,
                null,
                null,
                null,
                command.provider,
                null,
                null,
                "Recharge request with no recipient number"
            )
        }

        if(command.cardCode == null){
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.ALFA_RECHARGE_NO_CARD_CODE,
                null,
                null,
                0.02,
                command.provider,
                null,
                null,
                "credit transfer request with no card code"
            )
        }

        // Alfa recharge is an INTERACTIVE USSD session: dialing the menu string
        // (*111*3*2*{phone}*1*{code}#) lands on a confirmation dialog ("press 1 then YES") that
        // needs one more in-session input — which the headless TelephonyManager.sendUssdRequest
        // cannot answer, so it dies as "No USSD reply detected". Instead we dial via ACTION_CALL
        // to surface the system USSD dialog and let AlfaUssdAccessibilityService drive the
        // confirm step, then read the operator's terminal reply. This requires the accessibility
        // service to be enabled on the device.
        if (!AccessibilityUtil.isServiceEnabled(context, AlfaUssdAccessibilityService::class.java)) {
            return CommandResultDTO(
                command.commandId,
                command.recipientNumber,
                command.commandType,
                System.currentTimeMillis(),
                CommandResultCodes.ALFA_RECHARGE_ACCESSIBILITY_DISABLED,
                command.amount,
                null,
                null,
                command.provider,
                null,
                null,
                "Accessibility service not enabled; cannot complete interactive recharge"
            )
        }

        val subId = ProviderStore.alfaSim!!.subscriptionId
        val ussdCode = ProviderStore.alfaThirdPartyRechargeSmsTemplate
            .replace("{code}", command.cardCode)
            .replace("{phone}", command.recipientNumber)

        val deferred = CompletableDeferred<UssdSessionCoordinator.Outcome>()
        UssdSessionCoordinator.begin(
            UssdSessionCoordinator.Session(
                commandId = command.commandId,
                confirmDigits = listOf("1"),
                deferred = deferred,
            )
        )

        val powerManager = context.getSystemService(PowerManager::class.java)
        @Suppress("DEPRECATION")
        val wakeLock = powerManager?.newWakeLock(
            PowerManager.SCREEN_BRIGHT_WAKE_LOCK or PowerManager.ACQUIRE_CAUSES_WAKEUP,
            "MobileBridge:AlfaUssd"
        )

        val outcome = try {
            wakeLock?.acquire(90_000L)
            val dialTrampoline = Intent(context, UssdSessionActivity::class.java).apply {
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                putExtra(UssdSessionActivity.EXTRA_SUB_ID, subId)
                putExtra(UssdSessionActivity.EXTRA_USSD, ussdCode)
            }
            context.startActivity(dialTrampoline)
            withTimeoutOrNull(60_000L) { deferred.await() }
        } finally {
            // No-op if already resolved; otherwise clears the session + finishes the trampoline so
            // a stuck dialog can never wedge the serial command worker.
            UssdSessionCoordinator.timeout()
            if (wakeLock?.isHeld == true) wakeLock.release()
        }

        val rawReply = (outcome as? UssdSessionCoordinator.Outcome.Terminal)?.rawReply
        val lower = rawReply?.lowercase() ?: ""

        // Failure patterns are checked FIRST so wording like "recharge failed" can't be caught by
        // a loose success pattern. An unclassifiable reply is parked for manual review rather than
        // guessed as success — a false 4000 wrongly completes the order (unsafe), whereas a missed
        // success is merely a manual queue entry.
        return when {
            outcome == null || outcome is UssdSessionCoordinator.Outcome.Timeout ->
                CommandResultDTO(
                    command.commandId, command.recipientNumber, command.commandType,
                    System.currentTimeMillis(), CommandResultCodes.ALFA_RECHARGE_REPLY_TIMEOUT,
                    command.amount, null, null, command.provider, null, null,
                    "No interactive USSD reply detected"
                )
            outcome is UssdSessionCoordinator.Outcome.Error ->
                CommandResultDTO(
                    command.commandId, command.recipientNumber, command.commandType,
                    System.currentTimeMillis(), CommandResultCodes.ALFA_RECHARGE_DIAL_FAILED,
                    command.amount, null, null, command.provider, null, null,
                    "Could not drive USSD dialog: ${outcome.reason}"
                )
            ProviderStore.failureMatchPatterns.any { it.isNotBlank() && lower.contains(it.lowercase()) } ||
                lower.contains("fail") ->
                CommandResultDTO(
                    command.commandId, command.recipientNumber, command.commandType,
                    System.currentTimeMillis(), CommandResultCodes.ALFA_RECHARGE_PROVIDER_REJECTED,
                    command.amount, null, null, command.provider, null, null,
                    "Recharge rejected by provider: $rawReply", rawReply
                )
            ProviderStore.successMatchPatterns.any { it.isNotBlank() && lower.contains(it.lowercase()) } ->
                CommandResultDTO(
                    command.commandId, command.recipientNumber, command.commandType,
                    System.currentTimeMillis(), CommandResultCodes.ALFA_RECHARGE_SUCCESS,
                    command.amount, null, null, command.provider, null, null,
                    null, rawReply
                )
            else ->
                CommandResultDTO(
                    command.commandId, command.recipientNumber, command.commandType,
                    System.currentTimeMillis(), CommandResultCodes.ALFA_RECHARGE_REPLY_PARSE_FAILED,
                    command.amount, null, null, command.provider, null, null,
                    "Unrecognized operator reply; parked for manual review", rawReply
                )
        }
    }
}
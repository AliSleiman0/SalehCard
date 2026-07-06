package com.example.mobilebridgev2.sms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony

class SmsReplyReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        val messages = Telephony.Sms.Intents.getMessagesFromIntent(intent)

        for (sms in messages) {
            val sender = sms.originatingAddress ?: continue
            val body = sms.messageBody ?: continue

            SmsReplyRouter.onSmsReceived(sender, body)
        }
    }
}
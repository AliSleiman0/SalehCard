package com.example.mobilebridgev2.service


import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.telephony.SubscriptionManager
import androidx.annotation.RequiresApi
import androidx.annotation.RequiresPermission
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.core.content.PermissionChecker
import com.example.mobilebridgev2.CommandDispatcher
import com.example.mobilebridgev2.dto.CommandDTO
import com.example.mobilebridgev2.dto.ConfigurationDTO
import com.example.mobilebridgev2.ProviderStore
import com.example.mobilebridgev2.R
import com.example.mobilebridgev2.dto.CommandResultCodes
import com.example.mobilebridgev2.dto.CommandResultDTO
import com.example.mobilebridgev2.net.BridgeReporter
import com.example.mobilebridgev2.retrofit.RetrofitClient
import com.example.mobilebridgev2.sms.SmsReplyRouter
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.util.UUID

class BridgeForegroundService : Service() {

    private var engineStarted: Boolean = false
    private var configLoaded: Boolean = false
    private var turn: Boolean = false     //false: alfa, true: touch

    private val touchCommandsQueue = Channel<CommandDTO>(capacity = Channel.UNLIMITED)
    private val alfaCommandsQueue = Channel<CommandDTO>(capacity = Channel.UNLIMITED)

    companion object {
        private const val NOTIFICATION_ID = 100
        private const val CHANNEL_ID = "mobile_bridge_channel"
        private const val CHANNEL_NAME = "Mobile Bridge"
    }

    private val serviceScope = CoroutineScope(
        SupervisorJob() + Dispatchers.IO
    )

    override fun onCreate() {
        super.onCreate()
        com.example.mobilebridgev2.config.DeviceConfigStore.init(this)
        createNotificationChannel()
    }

    @RequiresApi(Build.VERSION_CODES.S)
    @RequiresPermission(Manifest.permission.CALL_PHONE)
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startAsForegroundService()

        if (hasRequiredPermissions()) {
            startBridgeEngine()
        } else {
            BridgeReporter.log("CRITICAL", "Required permissions are not granted")
            stopSelf()
        }

        return START_STICKY
    }

    private fun startAsForegroundService() {
        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle("MobileBridge is running")
            .setContentText("Polling commands and monitoring SIM balance")
            .setOngoing(true)
            .build()

        ServiceCompat.startForeground(
            this,
            100,
            notification,
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC
            } else {
                0
            }
        )
    }

    private fun hasRequiredPermissions(): Boolean {
        val requiredPermissions = mutableListOf(
            Manifest.permission.CALL_PHONE,
            Manifest.permission.SEND_SMS,
            Manifest.permission.READ_PHONE_STATE,
            Manifest.permission.RECEIVE_SMS,
            Manifest.permission.READ_SMS
        )

        return requiredPermissions.all { permission ->
            PermissionChecker.checkSelfPermission(this, permission) ==
                    PermissionChecker.PERMISSION_GRANTED
        }
    }

    @RequiresApi(Build.VERSION_CODES.S)
    @RequiresPermission(Manifest.permission.CALL_PHONE)
    private fun startBridgeEngine() {
        if (engineStarted) return
        engineStarted = true

        serviceScope.launch {
            try {
                loadProviderConfigAndState()
                readActiveSimSubscriptions()

                launch { startHeartbeatLoop() }
                launch { startPollingLoop() }
                launch { startCommandQueueWorker() }
                launch { regularBalanceChecks() }

            } catch (e: Exception) {
                e.printStackTrace()
                engineStarted = false
            }
        }
    }

    private suspend fun regularBalanceChecks() {
        while (true) {
            touchCommandsQueue.send(
                CommandDTO(
                    UUID.randomUUID().toString(),
                    "touch",
                    "CHECK_BALANCE",
                    System.currentTimeMillis(),
                    null,
                    null,
                    null,
                    null
                )
            )

            delay(30 * 60 * 1000L)
        }
    }

    /** Sends a liveness + SIM-balance heartbeat on the server-advertised cadence. */
    private suspend fun startHeartbeatLoop() {
        while (serviceScope.isActive) {
            BridgeReporter.heartbeat()
            delay(ProviderStore.heartbeatIntervalSeconds.coerceAtLeast(30) * 1000L)
        }
    }

    private suspend fun startPollingLoop() {
        while (serviceScope.isActive) {
            try {
                // Lease a small batch; the server hands out only this device's
                // operators and never re-leases a command already in flight.
                val commandList = RetrofitClient.api.getCommands(3)
                commandList.forEach { command ->
                    when (command.provider) {
                        "touch" -> touchCommandsQueue.send(command)
                        "alfa" -> alfaCommandsQueue.send(command)
                        else -> BridgeReporter.log("WARN", "Command with unknown provider: ${command.provider}")
                    }
                }
            } catch (e: Exception) {
                BridgeReporter.log("WARN", "Polling failed: ${e.message}")
            }
            delay(ProviderStore.pollIntervalSeconds.coerceAtLeast(2) * 1000L)
        }
    }

    @RequiresApi(Build.VERSION_CODES.S)
    @RequiresPermission(Manifest.permission.CALL_PHONE)
    private suspend fun startCommandQueueWorker() {
        while(serviceScope.coroutineContext.isActive){
            val command = if (turn) {
                touchCommandsQueue.tryReceive().getOrNull()
                    ?: alfaCommandsQueue.tryReceive().getOrNull()
            } else {
                alfaCommandsQueue.tryReceive().getOrNull()
                    ?: touchCommandsQueue.tryReceive().getOrNull()
            }

            if (command == null) {
                // Idle — no log spam; just wait for the next poll to enqueue work.
                delay(1000)
                continue
            }

            turn = !turn

            try {
                val result = CommandDispatcher.dispatch(command, applicationContext)
                BridgeReporter.reportResult(result)
            } catch (e: Exception) {
                BridgeReporter.reportResult(
                    CommandResultDTO(
                        id = command.commandId,
                        recipientNumber = command.recipientNumber,
                        type = command.commandType,
                        timestamp = System.currentTimeMillis(),
                        statusCode = CommandResultCodes.COMMAND_EXECUTION_FAILED,
                        amount = null,
                        cardCode = null,
                        billingAmount = null,
                        provider = command.provider,
                        balance = null,
                        validityDate = null,
                        errorMessage = "Error in command execution: ${e.message}"
                    )
                )
            }
        }
    }

    private fun fillProviderStore(dto: ConfigurationDTO) {
        ProviderStore.deviceId = dto.deviceId
        ProviderStore.touchBalanceCheckUssd = dto.touchBalanceCheckUssd
        ProviderStore.alfaBalanceCheckUssd = dto.alfaBalanceCheckUssd
        ProviderStore.touchThirdPartyRechargeTemplate = dto.touchThirdPartyRechargeTemplate
        ProviderStore.touchCreditTransferSmsTemplate = dto.touchCreditTransferSmsTemplate
        ProviderStore.touchCreditTransferDestination = dto.touchCreditTransferDestination
        ProviderStore.alfaThirdPartyRechargeSmsTemplate = dto.alfaThirdPartyRechargeSmsTemplate
        ProviderStore.alfaCreditTransferDestination = dto.alfaCreditTransferDestination
        ProviderStore.alfaThirdPartyRechargeDestination = dto.alfaThirdPartyRechargeDestination
        ProviderStore.touchMinimumAllowedBalance = dto.touchMinimumAllowedBalance
        ProviderStore.alfaMinimumAllowedBalance = dto.alfaMinimumAllowedBalance
        ProviderStore.touchSimBalance = dto.touchSimBalance
        ProviderStore.touchSimValidityDate = dto.touchSimValidityDate
        ProviderStore.touchCreditTransferMessageFee = dto.touchCreditTransferMessageFee
        ProviderStore.alfaCreditTransferMessageFee = dto.alfaCreditTransferMessageFee
        ProviderStore.alfaCreditTransferSmsTemplate = dto.alfaCreditTransferSmsTemplate
        // Alfa balance echo (restores state after a reinstall) + control knobs +
        // server-configurable reply matching.
        ProviderStore.alfaSimBalance = dto.alfaSimBalance
        ProviderStore.alfaSimValidityDate = dto.alfaSimValidityDate
        ProviderStore.pollIntervalSeconds = dto.pollIntervalSeconds
        ProviderStore.heartbeatIntervalSeconds = dto.heartbeatIntervalSeconds
        ProviderStore.maxSmsPerHalfHour = dto.maxSmsPerHalfHour
        ProviderStore.successMatchPatterns = dto.successMatchPatterns
        ProviderStore.failureMatchPatterns = dto.failureMatchPatterns
    }

    private fun readActiveSimSubscriptions() {
        val hasPermission =
            PermissionChecker.checkSelfPermission(
                this,
                Manifest.permission.READ_PHONE_STATE
            ) == PermissionChecker.PERMISSION_GRANTED

        if (!hasPermission) {
            BridgeReporter.log("CRITICAL", "READ_PHONE_STATE permission not granted")
            return
        }

        val subscriptionManager =
            getSystemService(SubscriptionManager::class.java)

        val sims = subscriptionManager.activeSubscriptionInfoList

        if (sims == null) {
            BridgeReporter.log("CRITICAL", "Device doesn't have any SIM cards")
            return
        }

        if (sims.size != 2) {
            BridgeReporter.log("WARN", "Device doesn't have dual SIM (found ${sims.size})")
        }

        sims.forEach { sim ->
            val name = "${sim.carrierName} ${sim.displayName}".lowercase()

            when {
                "alfa" in name -> {
                    ProviderStore.alfaSim = sim
                }

                "touch" in name || "mtc" in name -> {
                    ProviderStore.touchSim = sim
                }
            }
        }
    }

    private suspend fun loadProviderConfigAndState() {
        try {
            val dto = RetrofitClient.api.getConfig()
            fillProviderStore(dto)
            configLoaded = true
        } catch (e: Exception) {
            BridgeReporter.log("WARN", "Config not loaded: ${e.message}")
            configLoaded = false
        }
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                CHANNEL_NAME,
                NotificationManager.IMPORTANCE_LOW
            )

            val notificationManager =
                getSystemService(NotificationManager::class.java)

            notificationManager.createNotificationChannel(channel)
        }
    }

    override fun onDestroy() {
        engineStarted = false
        configLoaded = false

        SmsReplyRouter.cancelPendingReply()

        serviceScope.cancel("BridgeForegroundService destroyed")

        touchCommandsQueue.close()
        alfaCommandsQueue.close()

        stopForeground(STOP_FOREGROUND_REMOVE)

        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }
}
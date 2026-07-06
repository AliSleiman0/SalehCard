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
import com.example.mobilebridgev2.dto.LogErrorCodes
import com.example.mobilebridgev2.retrofit.RetrofitClient
import com.example.mobilebridgev2.sms.SmsReplyRouter
import com.example.mobilebridgev2.websocket.BridgeWebSocketClient
import com.example.mobilebridgev2.websocket.WebSocketLogDTO
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
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
        createNotificationChannel()
    }

    @RequiresApi(Build.VERSION_CODES.S)
    @RequiresPermission(Manifest.permission.CALL_PHONE)
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startAsForegroundService()

        if (hasRequiredPermissions()) {
            startBridgeEngine()
        } else {
            BridgeWebSocketClient.log(WebSocketLogDTO(
                level = "CRITICAL",
                errorCode = LogErrorCodes.REQUIRED_PERMISSIONS_NOT_GRANTED,
                message = "Required permissions are not granted",
                timestamp = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
            ))
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

                startWebSocketReporter()

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
                    "CHECK-BALANCE",
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

    private fun startWebSocketReporter() {
        BridgeWebSocketClient.connect(ProviderStore.deviceId.toString(),
            "hello world",
            "application/json")
    }

    private suspend fun startPollingLoop() {
        while(serviceScope.isActive){
            try{
                val commandList = RetrofitClient.api.getPendingCommands(ProviderStore.deviceId)
                commandList.forEach{ command ->
                    when(command.provider){
                        "touch" -> touchCommandsQueue.send(command)
                        "alfa" -> alfaCommandsQueue.send(command)
                        else -> BridgeWebSocketClient.log(WebSocketLogDTO(
                            level = "WARN",
                            errorCode = LogErrorCodes.INVALID_PROVIDER,
                            message = "Command sent with unknown provider",
                            timestamp = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
                        ))
                    }
                }
            }
            catch (e: Exception){
                BridgeWebSocketClient.log(WebSocketLogDTO(
                    level = "CRITICAL",
                    errorCode = LogErrorCodes.POLLING_LOOP_START_FAILED,
                    message = "Polling loop did not start: $e",
                    timestamp = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
                ))
            }
            delay(5000)
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
                BridgeWebSocketClient.log(WebSocketLogDTO(
                    level = "INFO",
                    errorCode = null,
                    message = "Command queues are empty",
                    timestamp = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
                ))
                delay(1000)
                continue
            }

            turn = !turn

            try{
                val result = CommandDispatcher.dispatch(command, applicationContext)
                if(!BridgeWebSocketClient.isConnected) BridgeWebSocketClient.connect(ProviderStore.deviceId.toString(),
                    "hello world",
                    "application/json")
                BridgeWebSocketClient.sendCommandResult(result)
            }
            catch (e: Exception){
                if(!BridgeWebSocketClient.isConnected) BridgeWebSocketClient.connect(ProviderStore.deviceId.toString(),
                    "hello world",
                    "application/json")
                BridgeWebSocketClient.sendCommandResult(CommandResultDTO(
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
                    errorMessage = "Error in command execution"
                ))
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
    }

    private fun readActiveSimSubscriptions() {
        val hasPermission =
            PermissionChecker.checkSelfPermission(
                this,
                Manifest.permission.READ_PHONE_STATE
            ) == PermissionChecker.PERMISSION_GRANTED

        if (!hasPermission) {
            BridgeWebSocketClient.log(WebSocketLogDTO(
                level = "CRITICAL",
                errorCode = LogErrorCodes.READ_PHONE_STATE_PERMISSION_NOT_GRANTED,
                message = "READ_PHONE_STATE permission not granted",
                timestamp = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
            ))
            return;
        }

        val subscriptionManager =
            getSystemService(SubscriptionManager::class.java)

        val sims = subscriptionManager.activeSubscriptionInfoList

        if(sims == null){
            BridgeWebSocketClient.sendMessage("Device doesn't have any SIM cards")
            return
        }

        if(sims.size != 2){
            BridgeWebSocketClient.sendMessage("Device doesn't have dual SIM")
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

    private suspend fun loadProviderConfigAndState(){
        try {
            val dto = RetrofitClient.api.getDeviceConfig()
            fillProviderStore(dto)
            configLoaded = true
        } catch (e: Exception) {
            if(!BridgeWebSocketClient.isConnected) BridgeWebSocketClient.connect(ProviderStore.deviceId.toString(),
                "hello world",
                "application/json")
            BridgeWebSocketClient.sendMessage("Config not loaded")
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

        BridgeWebSocketClient.disconnect()

        stopForeground(STOP_FOREGROUND_REMOVE)

        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }
}
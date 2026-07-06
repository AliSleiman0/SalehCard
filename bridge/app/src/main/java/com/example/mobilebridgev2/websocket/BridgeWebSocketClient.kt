package com.example.mobilebridgev2.websocket

import com.example.mobilebridgev2.ProviderStore
import com.example.mobilebridgev2.dto.CommandResultDTO
import com.google.gson.Gson
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.time.LocalDate
import java.time.format.DateTimeFormatter

object BridgeWebSocketClient {
    private val gson = Gson()
    private val client = OkHttpClient()

    private var webSocket: WebSocket ?= null
    var isConnected = false

    fun connect(
        deviceId: String,
        jwtToken: String,
        appType: String
    ) {
        val request = Request.Builder()
            .url("ws://10.0.2.2:3000/ws/bridge?deviceId=$deviceId")
            .addHeader("Authorization", "Bearer $jwtToken")
            .addHeader("App-Type", appType)
            .build()

        webSocket = client.newWebSocket(
            request,
            object : WebSocketListener() {

                override fun onOpen(webSocket: WebSocket, response: Response) {
                    isConnected = true
                    webSocket.send("connection established from device $deviceId")
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    val message = gson.fromJson(text, ReceivedMessageDTO::class.java)
                    if(message.topupObject != null){
                        ProviderStore.touchSimBalance += message.topupObject.addedBalance ?: 0.0
                        val formatter = DateTimeFormatter.ofPattern("dd-MM-yyyy")

                        val date = LocalDate.parse(ProviderStore.touchSimValidityDate, formatter)

                        val daysToAdd: Int = message.topupObject.addedValidityDays ?: 0

                        val newDate = date.plusDays(
                            daysToAdd.toLong()
                        )

                        ProviderStore.touchSimValidityDate = newDate.format(formatter)
                    }

                    val dto = message.configurationObject

                    if(dto != null) {
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
                }

                override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                    isConnected = false
                    webSocket.close(code, reason)
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    isConnected = false
                }

                override fun onFailure(
                    webSocket: WebSocket,
                    t: Throwable,
                    response: Response?
                ) {
                    isConnected = false
                    t.printStackTrace()
                }
            }
        )
    }

    fun sendCommandResult(result: CommandResultDTO): Boolean{
        val gsonResult = gson.toJson(result)
        if(!isConnected) connect(ProviderStore.deviceId.toString(),
            "hello world",
            "application/json")
        return webSocket?.send(gsonResult) ?: false
    }

    fun disconnect() {
        isConnected = false
        webSocket?.close(1000, "Bridge service stopped")
        webSocket = null
    }

    fun sendMessage(message: String) {
        if(!isConnected) connect(ProviderStore.deviceId.toString(),
            "hello world",
            "application/json")
        webSocket?.send(message)
    }

    fun log(log: WebSocketLogDTO){
        val gsonLog = gson.toJson(log)
        if(!isConnected) connect(ProviderStore.deviceId.toString(),
            "hello world",
            "application/json")
        webSocket?.send(gsonLog)
    }

    fun demandCredits(){
        if(!isConnected) connect(ProviderStore.deviceId.toString(),
            "hello world",
            "application/json")
        webSocket?.send("RECHARGE TOUCH LINE")
    }

}
package com.example.mobilebridgev2.websocket

import kotlin.time.Clock
import kotlin.time.ExperimentalTime

data class WebSocketLogDTO @OptIn(ExperimentalTime::class) constructor(
    val level: String,
    val errorCode: Int?,
    val message: String,
    val timestamp: String = Clock.System.now().toString()
)

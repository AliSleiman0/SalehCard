package com.example.mobilebridgev2.dto

data class CreditRequestDTO(
    val provider: String,
    val simNumber: String,
    val timestamp: Long,
    val requestedBalance: Double?,
    val requestedValidityDays: Int?
)

package com.example.mobilebridgev2.retrofit

import com.example.mobilebridgev2.dto.CommandDTO
import com.example.mobilebridgev2.dto.CommandResultDTO
import com.example.mobilebridgev2.dto.ConfigurationDTO
import com.example.mobilebridgev2.dto.CreditRequestDTO
import com.example.mobilebridgev2.dto.PermissionLogDTO
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface BridgeApi {

    @GET("config")
    suspend fun getDeviceConfig(): ConfigurationDTO

    @GET("commands/poll")
    suspend fun getPendingCommands(@Query("DeviceId") deviceId: Int): List<CommandDTO>

    @POST("commands/{id}/result")
    suspend fun reportCommandResult(@Path("id") commandId: String, @Body result: CommandResultDTO)

    @POST("permissions/{deviceId}")
    suspend fun reportPermissionChoice(@Path("deviceId") deviceId: String, @Body permissionChoice: PermissionLogDTO)

    @GET("configuration/{deviceId}/{provider}")
    suspend fun configure(@Path("deviceId") deviceId: String, @Path("provider") provider: String): ConfigurationDTO

    @POST("request/{deviceId}/{provider}")
    suspend fun requestCreditsForDevice(@Path("deviceId") deviceId: String, @Path("provider") provider: String, @Body creditRequest: CreditRequestDTO)
}
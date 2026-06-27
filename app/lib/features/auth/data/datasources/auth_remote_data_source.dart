import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../dtos/auth_response_dto.dart';

class AuthRemoteDataSource {
  const AuthRemoteDataSource(this._dio);

  final Dio _dio;

  Future<AuthResponseDto> login({
    required String email,
    required String password,
  }) async {
    final response = await _dio.post<dynamic>(
      '/auth/login',
      data: {'email': email, 'password': password},
    );
    return AuthResponseDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// Phone + password sign-in.
  Future<AuthResponseDto> loginByPhone({
    required String phone,
    required String password,
  }) async {
    final response = await _dio.post<dynamic>(
      '/auth/login-phone',
      data: {'phone': phone, 'password': password},
    );
    return AuthResponseDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// Requests an OTP code be sent to [phone] (E.164).
  Future<void> requestOtp({required String phone}) async {
    final response = await _dio.post<dynamic>(
      '/auth/otp/request',
      data: {'phone': phone},
    );
    unwrap(response);
  }

  /// Verifies an OTP code, authenticating (and creating the account on first
  /// sign-in). An optional [password] is set on a new account (signup).
  Future<AuthResponseDto> verifyOtp({
    required String phone,
    required String code,
    String? password,
    String? name,
  }) async {
    final response = await _dio.post<dynamic>(
      '/auth/otp/verify',
      data: {
        'phone': phone,
        'code': code,
        if (password != null && password.isNotEmpty) 'password': password,
        if (name != null && name.isNotEmpty) 'name': name,
      },
    );
    return AuthResponseDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }
}

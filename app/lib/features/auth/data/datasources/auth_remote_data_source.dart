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
}

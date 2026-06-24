import 'package:json_annotation/json_annotation.dart';

import 'user_dto.dart';

part 'auth_response_dto.g.dart';

/// Mirrors the backend AuthResponse. [refreshToken] is present because this
/// client sends `X-Client: mobile`.
@JsonSerializable()
class AuthResponseDto {
  const AuthResponseDto({
    required this.accessToken,
    required this.user,
    this.refreshToken,
  });

  final String accessToken;
  final String? refreshToken;
  final UserDto user;

  factory AuthResponseDto.fromJson(Map<String, dynamic> json) =>
      _$AuthResponseDtoFromJson(json);

  Map<String, dynamic> toJson() => _$AuthResponseDtoToJson(this);
}

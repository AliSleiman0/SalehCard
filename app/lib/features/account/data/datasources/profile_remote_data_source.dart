import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../../../auth/data/dtos/saved_player_id_dto.dart';
import '../../../auth/data/dtos/user_dto.dart';
import '../../../auth/domain/entities/saved_player_id.dart';

/// Talks to the profile endpoints. Reuses the shared auth [UserDto] (the
/// `/users/me` payload is the same User shape returned by login).
class ProfileRemoteDataSource {
  const ProfileRemoteDataSource(this._dio);

  final Dio _dio;

  /// GET /users/me → the authenticated user's profile.
  Future<UserDto> getMe() async {
    final response = await _dio.get<dynamic>('/users/me');
    return UserDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// PATCH /users/me → updates only the editable fields ([locale],
  /// [savedPlayerIds]) and returns the updated user. Sends only non-null fields
  /// so an omitted field is left untouched server-side.
  Future<UserDto> updateMe({
    String? locale,
    List<SavedPlayerId>? savedPlayerIds,
  }) async {
    final data = <String, dynamic>{};
    if (locale != null) data['locale'] = locale;
    if (savedPlayerIds != null) {
      data['savedPlayerIds'] = savedPlayerIds
          .map((e) => SavedPlayerIdDto.fromEntity(e).toJson())
          .toList();
    }
    final response = await _dio.patch<dynamic>('/users/me', data: data);
    return UserDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }
}

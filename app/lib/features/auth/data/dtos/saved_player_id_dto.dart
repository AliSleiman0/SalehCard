import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/saved_player_id.dart';

part 'saved_player_id_dto.g.dart';

@JsonSerializable()
class SavedPlayerIdDto {
  const SavedPlayerIdDto({required this.label, required this.value});

  final String label;
  final String value;

  factory SavedPlayerIdDto.fromJson(Map<String, dynamic> json) =>
      _$SavedPlayerIdDtoFromJson(json);

  Map<String, dynamic> toJson() => _$SavedPlayerIdDtoToJson(this);

  factory SavedPlayerIdDto.fromEntity(SavedPlayerId e) =>
      SavedPlayerIdDto(label: e.label, value: e.value);

  SavedPlayerId toEntity() => SavedPlayerId(label: label, value: value);
}

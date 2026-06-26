import 'package:json_annotation/json_annotation.dart';

import '../../../../core/i18n/i18n_string.dart';
import '../../domain/entities/product.dart';

part 'product_dto.g.dart';

@JsonSerializable()
class VariantDto {
  const VariantDto({
    required this.id,
    this.denomination = '',
    this.price = 0,
  });

  final String id;
  final String denomination;
  final double price;

  factory VariantDto.fromJson(Map<String, dynamic> json) =>
      _$VariantDtoFromJson(json);

  Map<String, dynamic> toJson() => _$VariantDtoToJson(this);

  Variant toEntity() =>
      Variant(id: id, denomination: denomination, price: price);
}

@JsonSerializable()
class RatingsDto {
  const RatingsDto({this.average = 0, this.count = 0});

  final double average;
  final int count;

  factory RatingsDto.fromJson(Map<String, dynamic> json) =>
      _$RatingsDtoFromJson(json);

  Map<String, dynamic> toJson() => _$RatingsDtoToJson(this);
}

@JsonSerializable()
class InputFieldConstraintsDto {
  const InputFieldConstraintsDto({this.min, this.max, this.options});

  final double? min;
  final double? max;
  final List<String>? options;

  factory InputFieldConstraintsDto.fromJson(Map<String, dynamic> json) =>
      _$InputFieldConstraintsDtoFromJson(json);

  Map<String, dynamic> toJson() => _$InputFieldConstraintsDtoToJson(this);

  InputFieldConstraints toEntity() =>
      InputFieldConstraints(min: min, max: max, options: options);
}

@JsonSerializable(explicitToJson: true)
class InputFieldDto {
  const InputFieldDto({
    required this.key,
    this.label,
    this.type = 'text',
    this.constraints,
    this.sensitive = false,
  });

  final String key;
  final Map<String, dynamic>? label;
  final String type;
  final InputFieldConstraintsDto? constraints;
  final bool sensitive;

  factory InputFieldDto.fromJson(Map<String, dynamic> json) =>
      _$InputFieldDtoFromJson(json);

  Map<String, dynamic> toJson() => _$InputFieldDtoToJson(this);

  InputField toEntity() => InputField(
        key: key,
        label: I18nString.fromJson(label),
        type: type,
        constraints: constraints?.toEntity(),
        sensitive: sensitive,
      );
}

@JsonSerializable(explicitToJson: true)
class ProductDto {
  const ProductDto({
    required this.id,
    this.title,
    this.category,
    this.images,
    this.variants,
    this.stock,
    this.available,
    this.fulfillmentType,
    this.inputFields,
    this.ratings,
  });

  final String id;
  final Map<String, dynamic>? title;
  final String? category;
  final List<String>? images;
  final List<VariantDto>? variants;
  final int? stock;
  final bool? available;
  final String? fulfillmentType;
  final List<InputFieldDto>? inputFields;
  final RatingsDto? ratings;

  factory ProductDto.fromJson(Map<String, dynamic> json) =>
      _$ProductDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ProductDtoToJson(this);

  Product toEntity() => Product(
        id: id,
        title: I18nString.fromJson(title),
        category: category ?? '',
        images: images ?? const [],
        variants:
            (variants ?? const []).map((v) => v.toEntity()).toList(),
        stock: stock ?? 0,
        available: available ?? false,
        fulfillmentType: fulfillmentType ?? 'code',
        inputFields:
            (inputFields ?? const []).map((f) => f.toEntity()).toList(),
        rating: ratings?.average,
        ratingCount: ratings?.count ?? 0,
      );
}

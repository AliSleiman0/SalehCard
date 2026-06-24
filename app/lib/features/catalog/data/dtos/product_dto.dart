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
    this.ratings,
  });

  final String id;
  final Map<String, dynamic>? title;
  final String? category;
  final List<String>? images;
  final List<VariantDto>? variants;
  final int? stock;
  final bool? available;
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
        rating: ratings?.average,
        ratingCount: ratings?.count ?? 0,
      );
}

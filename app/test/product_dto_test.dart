import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/catalog/data/dtos/product_dto.dart';

void main() {
  group('ProductDto.toEntity', () {
    test('maps backend JSON to a domain Product', () {
      final dto = ProductDto.fromJson(<String, dynamic>{
        'id': 'p1',
        'title': {'en': 'Gift Card', 'ar': 'بطاقة هدية', 'tr': 'Hediye'},
        'category': 'games',
        'images': ['https://example.com/a.png'],
        'variants': [
          {'id': 'v1', 'denomination': 'USD 5', 'price': 5},
          {'id': 'v2', 'denomination': 'USD 10', 'price': 10.5},
        ],
        'stock': 7,
        'available': true,
        'ratings': {'average': 4.5, 'count': 12},
      });

      final product = dto.toEntity();

      expect(product.id, 'p1');
      expect(product.title.resolve('ar'), 'بطاقة هدية');
      expect(product.title.resolve('en'), 'Gift Card');
      expect(product.category, 'games');
      expect(product.images.single, 'https://example.com/a.png');
      expect(product.variants.length, 2);
      // price arrives as an int in JSON; must become a double.
      expect(product.variants.first.price, 5.0);
      expect(product.fromPrice, 5.0);
      expect(product.stock, 7);
      expect(product.available, isTrue);
      expect(product.rating, 4.5);
      expect(product.ratingCount, 12);
    });

    test('tolerates missing optional fields', () {
      final dto = ProductDto.fromJson(<String, dynamic>{'id': 'p2'});
      final product = dto.toEntity();

      expect(product.id, 'p2');
      expect(product.title.resolve('en'), '');
      expect(product.variants, isEmpty);
      expect(product.fromPrice, isNull);
      expect(product.available, isFalse);
      expect(product.stock, 0);
    });

    test('parses live-offer enrichment (variant offerPrice + product offer)', () {
      final dto = ProductDto.fromJson(<String, dynamic>{
        'id': 'p3',
        'variants': [
          {'id': 'v1', 'denomination': 'USD 10', 'price': 10, 'offerPrice': 7},
          {'id': 'v2', 'denomination': 'USD 20', 'price': 20, 'offerPrice': 14},
        ],
        'available': true,
        'offer': {
          'discountType': 'percent',
          'discountValue': 30,
          'originalFromPrice': 10,
          'offerFromPrice': 7,
        },
      });

      final product = dto.toEntity();

      expect(product.hasOffer, isTrue);
      expect(product.offer!.discountLabel, '-30%');
      expect(product.offerFromPrice, 7.0);
      expect(product.fromPrice, 10.0);
      // per-variant discounted prices flow through
      expect(product.variants.first.offerPrice, 7.0);
      expect(product.variants.first.effectivePrice, 7.0);
      expect(product.variants.first.hasOffer, isTrue);
      expect(product.variants[1].offerPrice, 14.0);
    });

    test('no offer block leaves product/variants at base price', () {
      final dto = ProductDto.fromJson(<String, dynamic>{
        'id': 'p4',
        'variants': [
          {'id': 'v1', 'denomination': 'USD 5', 'price': 5},
        ],
        'available': true,
      });

      final product = dto.toEntity();

      expect(product.hasOffer, isFalse);
      expect(product.offer, isNull);
      expect(product.offerFromPrice, isNull);
      expect(product.variants.first.offerPrice, isNull);
      expect(product.variants.first.effectivePrice, 5.0);
      expect(product.variants.first.hasOffer, isFalse);
    });
  });

  group('Product image getters (thumbnail / thumbUrl / imageUrl)', () {
    test('thumbnail is parsed and thumbUrl prefers it', () {
      final product = ProductDto.fromJson(<String, dynamic>{
        'id': 'p5',
        'images': ['https://cdn/x.jpg'],
        'thumbnail': 'https://cdn/x_thumb.jpg',
      }).toEntity();

      expect(product.thumbnail, 'https://cdn/x_thumb.jpg');
      expect(product.thumbUrl, 'https://cdn/x_thumb.jpg');
      expect(product.imageUrl, 'https://cdn/x.jpg');
    });

    test('thumbUrl falls back to the display image when no thumbnail', () {
      final product = ProductDto.fromJson(<String, dynamic>{
        'id': 'p6',
        'images': ['https://cdn/x.jpg'],
      }).toEntity();

      expect(product.thumbnail, isNull);
      expect(product.thumbUrl, 'https://cdn/x.jpg');
    });

    test('empty-string thumbnail falls back to the display image (not "")', () {
      final product = ProductDto.fromJson(<String, dynamic>{
        'id': 'p7',
        'images': ['https://cdn/x.jpg'],
        'thumbnail': '',
      }).toEntity();

      expect(product.thumbUrl, 'https://cdn/x.jpg');
    });

    test('empty-string image yields a null imageUrl and thumbUrl', () {
      final product = ProductDto.fromJson(<String, dynamic>{
        'id': 'p8',
        'images': [''],
      }).toEntity();

      expect(product.imageUrl, isNull);
      expect(product.thumbUrl, isNull);
    });

    test('no images at all leaves both getters null', () {
      final product = ProductDto.fromJson(<String, dynamic>{'id': 'p9'}).toEntity();

      expect(product.imageUrl, isNull);
      expect(product.thumbUrl, isNull);
    });
  });
}

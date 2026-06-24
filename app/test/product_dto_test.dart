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
  });
}

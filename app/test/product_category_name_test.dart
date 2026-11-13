import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/catalog/data/dtos/product_dto.dart';

void main() {
  group('ProductDto categoryName', () {
    test('maps the assigned node name for display', () {
      final p = ProductDto.fromJson(<String, dynamic>{
        'id': 'p1',
        'category': 'cat-447',
        'categoryName': {'en': 'Dahab Transfer', 'ar': 'تحويل ذهب', 'tr': ''},
      }).toEntity();

      expect(p.category, 'cat-447'); // raw slug kept for filtering only
      expect(p.categoryName?.resolve('ar'), 'تحويل ذهب');
      expect(p.categoryName?.resolve('tr'), 'Dahab Transfer');
    });

    test('is null when the API sends none (no slug fallback)', () {
      final p = ProductDto.fromJson(<String, dynamic>{
        'id': 'p1',
        'category': 'cat-447',
      }).toEntity();

      expect(p.categoryName, isNull);
    });
  });
}

import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/browse/data/dtos/category_dto.dart';

void main() {
  group('CategoryDto.toEntity', () {
    test('maps a root-domain category with a count', () {
      final dto = CategoryDto.fromJson(<String, dynamic>{
        'id': 'c1',
        'legacyId': 10,
        'slug': 'games',
        'name': {'en': 'Games', 'ar': 'ألعاب', 'tr': 'Oyunlar'},
        'image': 'https://example.com/games.png',
        'rootDomain': 'games',
        'depth': 0,
        'productCount': 178,
      });

      final category = dto.toEntity();

      expect(category.id, 'c1');
      expect(category.slug, 'games');
      expect(category.name.resolve('ar'), 'ألعاب');
      expect(category.name.resolve('en'), 'Games');
      expect(category.image, 'https://example.com/games.png');
      expect(category.rootDomain, 'games');
      expect(category.depth, 0);
      expect(category.productCount, 178);
    });

    test('tolerates missing optional fields (no count)', () {
      final dto = CategoryDto.fromJson(<String, dynamic>{'id': 'c2'});
      final category = dto.toEntity();

      expect(category.id, 'c2');
      expect(category.name.resolve('en'), '');
      expect(category.productCount, isNull);
      expect(category.image, isNull);
      expect(category.depth, 0);
    });
  });
}

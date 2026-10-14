import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/config/regulated_finance.dart';
import 'package:salehcard_app/features/browse/data/datasources/category_remote_data_source.dart';
import 'package:salehcard_app/features/browse/data/dtos/category_dto.dart';
import 'package:salehcard_app/features/browse/data/repositories/category_repository_impl.dart';

class _FakeCategoryRemote extends CategoryRemoteDataSource {
  _FakeCategoryRemote() : super(Dio());

  @override
  Future<List<CategoryDto>> getCategories({
    bool withCounts = false,
    int? depth,
    String? rootDomain,
    String? parentId,
  }) async =>
      const [
        CategoryDto(id: '1', slug: 'games', rootDomain: 'games'),
        CategoryDto(id: '2', slug: 'crypto', rootDomain: 'wallets_crypto'),
        CategoryDto(id: '3', slug: 'transfers', rootDomain: 'money_transfers'),
      ];
}

Future<List<String>> _slugs() async {
  final result =
      await CategoryRepositoryImpl(_FakeCategoryRemote()).listCategories();
  return result.match((_) => [], (cats) => [for (final c in cats) c.slug]);
}

void main() {
  tearDown(() => debugDefaultTargetPlatformOverride = null);

  test('iOS hides crypto and money-transfer categories', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    expect(hideRegulatedFinance, isTrue);
    expect(await _slugs(), ['games']);
  });

  test('Android keeps every category', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    expect(hideRegulatedFinance, isFalse);
    expect(await _slugs(), ['games', 'crypto', 'transfers']);
  });

  test('crypto top-up method names are recognised', () {
    for (final name in ['USDT (TRC20)', 'Binance Pay', 'Crypto', 'Tether']) {
      expect(isCryptoTopUpMethodName(name), isTrue, reason: name);
    }
    for (final name in ['Whish Money', 'OMT', 'Cash at office']) {
      expect(isCryptoTopUpMethodName(name), isFalse, reason: name);
    }
  });
}

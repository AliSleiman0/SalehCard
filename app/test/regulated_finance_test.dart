import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/config/regulated_finance.dart';
import 'package:salehcard_app/core/network/paged.dart';
import 'package:salehcard_app/features/browse/data/datasources/category_remote_data_source.dart';
import 'package:salehcard_app/features/browse/data/dtos/category_dto.dart';
import 'package:salehcard_app/features/browse/data/repositories/category_repository_impl.dart';
import 'package:salehcard_app/features/catalog/data/datasources/catalog_remote_data_source.dart';
import 'package:salehcard_app/features/catalog/data/dtos/product_dto.dart';
import 'package:salehcard_app/features/catalog/data/repositories/catalog_repository_impl.dart';

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
        CategoryDto(id: '4', slug: 'alfa', rootDomain: 'telecom'),
        CategoryDto(id: '5', slug: 'cards', rootDomain: 'giftcards'),
        CategoryDto(id: '6', slug: 'orphan'),
      ];
}

/// Returns a fixed mixed-domain product list and records the rootDomain each
/// query was sent with.
class _FakeCatalogRemote extends CatalogRemoteDataSource {
  _FakeCatalogRemote() : super(Dio());

  final sentRootDomains = <String?>[];

  static const _products = [
    ProductDto(id: 'p1', category: 'alfa', rootDomain: 'telecom'),
    ProductDto(id: 'p2', category: 'pubg', rootDomain: 'games'),
    ProductDto(id: 'p3', category: 'alfa'), // no rootDomain → by slug
    ProductDto(id: 'p4', category: 'pubg'), // no rootDomain → by slug
    ProductDto(
      id: 'p5',
      category: 'alfa',
      rootDomain: 'telecom',
      fulfillmentType: 'transfer',
    ),
  ];

  @override
  Future<List<ProductDto>> getProducts({
    int page = 1,
    int limit = 20,
    String? category,
    String? categoryId,
    String? rootDomain,
    String? search,
    bool directOnly = false,
  }) async {
    sentRootDomains.add(rootDomain);
    return _products;
  }

  @override
  Future<Paged<ProductDto>> getProductsPage({
    int page = 1,
    int limit = 20,
    String? category,
    String? categoryId,
    String? rootDomain,
    String? search,
    bool directOnly = false,
  }) async {
    sentRootDomains.add(rootDomain);
    return const Paged(items: _products, total: 5);
  }

  @override
  Future<ProductDto> getProduct(String id) async =>
      _products.firstWhere((p) => p.id == id);
}

class _FakeFilter extends RegulatedCatalogFilter {
  _FakeFilter() : super(Dio());

  @override
  Future<Set<String>> allowedCategorySlugs() async => {'alfa'};
}

Future<List<String>> _categorySlugs() async {
  final result =
      await CategoryRepositoryImpl(_FakeCategoryRemote()).listCategories();
  return result.match((_) => [], (cats) => [for (final c in cats) c.slug]);
}

void main() {
  tearDown(() => debugDefaultTargetPlatformOverride = null);

  group('iOS (restricted to mobile recharge)', () {
    setUp(() => debugDefaultTargetPlatformOverride = TargetPlatform.iOS);

    test('scope is telecom and fails closed on unknown domains', () {
      expect(restrictToRealWorldServices, isTrue);
      expect(hideRegulatedFinance, isTrue);
      expect(catalogRootDomainScope, 'telecom');
      expect(isRootDomainAllowed('telecom'), isTrue);
      for (final d in ['games', 'giftcards', 'app_topups', '', null]) {
        expect(isRootDomainAllowed(d), isFalse, reason: '$d');
      }
    });

    test('only telecom categories are listed', () async {
      expect(await _categorySlugs(), ['alfa']);
    });

    test('product queries are pinned to telecom and filtered', () async {
      final remote = _FakeCatalogRemote();
      final repo = CatalogRepositoryImpl(remote, _FakeFilter());

      final list = await repo.getProducts(search: 'pubg');
      expect(list.match((_) => <String>[], (ps) => [for (final p in ps) p.id]),
          ['p1', 'p3']);

      final page = await repo.getProductsPage();
      expect(
        page.match((_) => <String>[], (pg) => [for (final p in pg.items) p.id]),
        ['p1', 'p3'],
      );
      expect(remote.sentRootDomains, ['telecom', 'telecom']);
    });

    test('asking for another domain returns nothing without a request',
        () async {
      final remote = _FakeCatalogRemote();
      final repo = CatalogRepositoryImpl(remote, _FakeFilter());
      final page = await repo.getProductsPage(rootDomain: 'games');
      expect(page.match((_) => -1, (pg) => pg.items.length), 0);
      expect(remote.sentRootDomains, isEmpty);
    });

    test('a hidden product opens as not found', () async {
      final repo = CatalogRepositoryImpl(_FakeCatalogRemote(), _FakeFilter());
      expect((await repo.getProduct('p1')).isRight(), isTrue);
      expect((await repo.getProduct('p2')).isLeft(), isTrue);
      expect((await repo.getProduct('p4')).isLeft(), isTrue);
    });

    test('category predicate allows only telecom slugs', () async {
      final allowed = await _FakeFilter().categoryAllowed();
      expect(allowed('alfa'), isTrue);
      expect(allowed('pubg'), isFalse);
      expect(allowed(null), isFalse);
    });

    test('crypto and Whish top-up methods are hidden', () {
      for (final name in ['USDT (TRC20)', 'Binance Pay', 'Whish Money']) {
        expect(isTopUpMethodHiddenOnRestricted(name), isTrue, reason: name);
      }
      for (final name in ['OMT', 'Cash at office', 'Bank transfer']) {
        expect(isTopUpMethodHiddenOnRestricted(name), isFalse, reason: name);
      }
    });
  });

  group('Android (full catalog)', () {
    setUp(() => debugDefaultTargetPlatformOverride = TargetPlatform.android);

    test('nothing is restricted', () async {
      expect(restrictToRealWorldServices, isFalse);
      expect(hideRegulatedFinance, isFalse);
      expect(catalogRootDomainScope, isNull);
      expect(isRootDomainAllowed('games'), isTrue);
      expect(await _categorySlugs(),
          ['games', 'crypto', 'transfers', 'alfa', 'cards', 'orphan']);
    });

    test('product queries pass through untouched', () async {
      final remote = _FakeCatalogRemote();
      final repo = CatalogRepositoryImpl(remote, _FakeFilter());
      final list = await repo.getProducts(rootDomain: 'games');
      expect(list.match((_) => -1, (ps) => ps.length), 5);
      expect(remote.sentRootDomains, ['games']);
      expect((await _FakeFilter().categoryAllowed())('pubg'), isTrue);
    });
  });

  test('method kinds are an allow-list on restricted builds', () {
    for (final kind in ['bank', 'cash', 'other']) {
      expect(isTopUpMethodShownOnRestricted(kind: kind, name: 'Sham Cash'),
          isTrue,
          reason: kind);
      // Defense in depth: a mis-set Type never exposes a crypto/Whish rail.
      expect(isTopUpMethodShownOnRestricted(kind: kind, name: 'USDT TRC20'),
          isFalse,
          reason: kind);
      expect(isTopUpMethodShownOnRestricted(kind: kind, name: 'Whish Money'),
          isFalse,
          reason: kind);
    }
    // Word boundaries: ordinary words are not crypto.
    expect(isCryptoTopUpMethodName('Automatic bank transfer'), isFalse);
    expect(isCryptoTopUpMethodName('Polygon (MATIC)'), isTrue);
    for (final kind in ['crypto', 'wallet_app', 'some_future_kind']) {
      expect(isTopUpMethodShownOnRestricted(kind: kind, name: 'Bank'), isFalse,
          reason: kind);
    }
    // Unclassified (legacy) methods fall back to the name deny-list.
    expect(isTopUpMethodShownOnRestricted(kind: '', name: 'OMT'), isTrue);
    expect(isTopUpMethodShownOnRestricted(kind: ' ', name: 'ETH (ERC20)'),
        isFalse);
    expect(isTopUpMethodShownOnRestricted(kind: '', name: 'Whish Money'),
        isFalse);
  });

  test('crypto names missed by the old regex are now recognised', () {
    for (final name in [
      'ETH (ERC20)',
      'Ethereum',
      'TRON / TRX',
      'Solana',
      'USDC',
      'Litecoin LTC',
      'BNB Smart Chain',
      'عملات رقمية',
    ]) {
      expect(isCryptoTopUpMethodName(name), isTrue, reason: name);
    }
    for (final name in ['Electronic transfer', 'Sham Cash', 'Western Union']) {
      expect(isCryptoTopUpMethodName(name), isFalse, reason: name);
    }
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

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../network/api_envelope.dart';
import '../network/providers.dart';

/// Whether this build hides crypto and money-transfer features. The App Store
/// only accepts those from developers enrolled as an organization, and the iOS
/// app ships from an individual account — so iOS hides them entirely while
/// Android (published under the Entrizo org account) keeps them.
bool get hideRegulatedFinance =>
    !kIsWeb && defaultTargetPlatform == TargetPlatform.iOS;

/// Root domains whose categories and products are hidden when
/// [hideRegulatedFinance] is on.
const regulatedRootDomains = {'wallets_crypto', 'money_transfers'};

/// Manual top-up methods that are crypto rails (matched on the admin-defined
/// method name, which is the only identifying field the API returns).
final _cryptoMethodName = RegExp(
  r'usdt|usdc|tether|crypto|bitcoin|\bbtc\b|binance|trc20|bep20',
  caseSensitive: false,
);

bool isCryptoTopUpMethodName(String name) => _cryptoMethodName.hasMatch(name);

/// Resolves the category slugs belonging to [regulatedRootDomains] so products
/// and offers (which only carry their category slug) can be filtered out. The
/// lookup is cached once it succeeds; a failure is not cached so the next call
/// retries.
class RegulatedCatalogFilter {
  RegulatedCatalogFilter(this._dio);

  final Dio _dio;
  Future<Set<String>>? _slugs;

  Future<Set<String>> blockedCategorySlugs() {
    if (!hideRegulatedFinance) return Future.value(const {});
    return _slugs ??= _load().catchError((Object error) {
      _slugs = null;
      throw error;
    });
  }

  Future<Set<String>> _load() async {
    final slugs = <String>{};
    for (final domain in regulatedRootDomains) {
      final response = await _dio.get<dynamic>(
        '/categories',
        queryParameters: {'rootDomain': domain},
      );
      for (final item in unwrap(response) as List<dynamic>) {
        final slug = (item as Map<String, dynamic>)['slug']?.toString();
        if (slug != null && slug.isNotEmpty) slugs.add(slug);
      }
    }
    return slugs;
  }
}

final regulatedCatalogFilterProvider = Provider<RegulatedCatalogFilter>(
  (ref) => RegulatedCatalogFilter(ref.watch(dioProvider)),
);

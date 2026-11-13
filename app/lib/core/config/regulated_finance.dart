import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../network/api_envelope.dart';
import '../network/providers.dart';

/// Debug-only switch to exercise the iOS store scope on Android/emulators:
/// `flutter run --dart-define=FORCE_IOS_SCOPE=true`. Compiled out of release
/// builds (kDebugMode is const false there), so it can never change what a
/// shipped build shows.
const _forceIosScope = kDebugMode && bool.fromEnvironment('FORCE_IOS_SCOPE');

/// Whether this build is limited to real-world services (mobile recharge).
///
/// App Review (guideline 3.1.1) does not allow the iOS app to sell game
/// currency, codes or gift cards used in other apps/platforms without In-App
/// Purchase, and the iOS app ships from an individual account that cannot offer
/// crypto or money transfer. So iOS shows only [iosAllowedRootDomains] —
/// prepaid mobile airtime, consumed on the user's SIM. Android is unchanged.
///
/// Deliberately platform-based, never a remote flag: changing what a reviewed
/// build shows from the server would breach guideline 2.3.1.
bool get restrictToRealWorldServices =>
    !kIsWeb && (defaultTargetPlatform == TargetPlatform.iOS || _forceIosScope);

/// Whether this build hides crypto and money-transfer features (USDT payment,
/// crypto top-up methods). True wherever the store scope is restricted.
bool get hideRegulatedFinance => restrictToRealWorldServices;

/// The only catalog root domains shown when [restrictToRealWorldServices].
const iosAllowedRootDomains = {'telecom'};

/// Root domain every catalog query is pinned to on restricted builds, so the
/// server paginates and counts only allowed products; null = unrestricted.
String? get catalogRootDomainScope =>
    restrictToRealWorldServices ? iosAllowedRootDomains.single : null;

/// Whether a category/product in [rootDomain] may be shown. Fails closed on
/// restricted builds: an unknown or empty domain is hidden.
bool isRootDomainAllowed(String? rootDomain) =>
    !restrictToRealWorldServices || iosAllowedRootDomains.contains(rootDomain);

/// Manual top-up methods that are crypto rails, matched on the admin-defined
/// method name. Only the fallback for methods with no `kind` yet — see
/// [isTopUpMethodShownOnRestricted].
final _cryptoMethodName = RegExp(
  r'usd[ct]|tether|crypto|bitcoin|\bbtc\b|binance|\bbnb\b|\bbsc\b|trc-?20|'
  r'bep-?20|erc-?20|ethereum|\beth\b|\btron\b|\btrx\b|solana|\bsol\b|'
  r'litecoin|\bltc\b|\bpolygon\b|\bmatic\b|\bdoge(coin)?\b|wallet address|'
  r'كريبتو|بيتكوين|تيثر|عملات رقمية',
  caseSensitive: false,
);

bool isCryptoTopUpMethodName(String name) => _cryptoMethodName.hasMatch(name);

/// Third-party payment-app rails not offered on restricted builds yet.
final _restrictedMethodName = RegExp(r'whish', caseSensitive: false);

/// Whether a manual top-up method is hidden on restricted builds: crypto rails
/// plus payment apps not yet integrated there.
bool isTopUpMethodHiddenOnRestricted(String name) =>
    isCryptoTopUpMethodName(name) || _restrictedMethodName.hasMatch(name);

/// Method kinds a restricted build may offer. An ALLOW-list: crypto, wallet_app
/// and any kind added later are hidden by default.
const restrictedAllowedMethodKinds = {'bank', 'cash', 'other'};

/// Whether a manual top-up method is shown on restricted builds. A classified
/// method (admin-set `kind`) is shown only if its kind is allow-listed; an
/// unclassified one (legacy, empty `kind`) falls back to the name deny-list so
/// existing methods keep working until an admin sets their kind.
bool isTopUpMethodShownOnRestricted({
  required String kind,
  required String name,
}) {
  final k = kind.trim();
  // Defense in depth: an allow-listed kind must ALSO pass the name deny-list,
  // so one mis-set Type (a USDT method marked "Other") can never expose a
  // crypto or Whish rail to App Review.
  if (k.isNotEmpty) {
    return restrictedAllowedMethodKinds.contains(k) &&
        !isTopUpMethodHiddenOnRestricted(name);
  }
  return !isTopUpMethodHiddenOnRestricted(name);
}

/// Resolves which category slugs are allowed on restricted builds, so items
/// that only carry a category slug (offers, order lines) can be filtered. The
/// lookup is cached once it succeeds; a failure is not cached so the next call
/// retries (and the caller fails closed).
class RegulatedCatalogFilter {
  RegulatedCatalogFilter(this._dio);

  final Dio _dio;
  Future<Set<String>>? _slugs;

  /// Predicate over a category slug: always true on unrestricted builds,
  /// otherwise true only for slugs under [iosAllowedRootDomains].
  Future<bool Function(String? slug)> categoryAllowed() async {
    if (!restrictToRealWorldServices) return (_) => true;
    final allowed = await allowedCategorySlugs();
    return (slug) => slug != null && allowed.contains(slug);
  }

  Future<Set<String>> allowedCategorySlugs() =>
      _slugs ??= _load().catchError((Object error) {
        _slugs = null;
        throw error;
      });

  Future<Set<String>> _load() async {
    final slugs = <String>{};
    for (final domain in iosAllowedRootDomains) {
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

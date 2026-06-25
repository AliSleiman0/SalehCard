import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../i18n/arb/app_localizations.dart';
import '../locale/locale_controller.dart';
import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';
import 'brand_logo.dart';

/// Shared grey top bar for the auth screens: an optional back button (start),
/// the centered brand mark, and a language toggle (end). All directional, so it
/// mirrors correctly under RTL, and theme-aware via [AppColors].
class AuthHeader extends ConsumerWidget {
  const AuthHeader({super.key, this.onBack});

  /// When non-null, shows a back chevron on the leading (start) edge.
  final VoidCallback? onBack;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    return Container(
      width: double.infinity,
      height: 64,
      color: colors.topbar,
      child: Stack(
        alignment: Alignment.center,
        children: [
          const BrandLogo(size: 46),
          if (onBack != null)
            PositionedDirectional(
              start: 12,
              top: 0,
              bottom: 0,
              child: IconButton(
                onPressed: onBack,
                icon: Icon(Icons.arrow_back_ios_new,
                    size: 18, color: colors.text),
              ),
            ),
          PositionedDirectional(
            end: 8,
            top: 0,
            bottom: 0,
            child: TextButton(
              onPressed: () =>
                  ref.read(localeControllerProvider.notifier).toggle(),
              style: TextButton.styleFrom(foregroundColor: AppTokens.brand1),
              child: Text(l10n.languageToggle),
            ),
          ),
        ],
      ),
    );
  }
}

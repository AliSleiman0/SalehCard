import 'package:flutter/material.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/widgets/app_spinner.dart';
import '../../../core/widgets/brand_logo.dart';

/// Branded launch screen shown while the session is restored
/// ([AuthStatus.unknown]). Continues the native splash (same dark background +
/// lockup) so the handoff into Flutter is seamless, then the router redirects to
/// `/home` or `/login` once auth resolves.
class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: context.colors.bg,
      body: const Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            BrandLogo(size: 120, showWordmark: true),
            SizedBox(height: 32),
            AppSpinner(),
          ],
        ),
      ),
    );
  }
}

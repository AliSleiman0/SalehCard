import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/config/app_config.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/theme_controller.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../widgets/delete_account_sheet.dart';

/// Menu / settings tab — foundation version: language + theme toggles and
/// logout. Expanded into the full designed menu + profile in the Account session.
class AccountMenuScreen extends ConsumerWidget {
  const AccountMenuScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final isAr = ref.watch(localeControllerProvider).languageCode == 'ar';
    final isDark = ref.watch(themeControllerProvider) == ThemeMode.dark;
    final user = ref.watch(authControllerProvider).user;
    final name = user?.name ?? '';
    final email = user?.email ?? '';
    // Prefer the name; fall back to phone for phone-OTP accounts without email.
    final identity = name.isNotEmpty ? name : email;
    final subtitle = name.isNotEmpty ? (email.isNotEmpty ? email : (user?.phone ?? '')) : '';

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(title: Text(l10n.navMenu)),
      body: ListView(
        children: [
          if (identity.isNotEmpty)
            ListTile(
              leading: const Icon(Icons.person_outline_rounded),
              title: Text(identity),
              subtitle: subtitle.isNotEmpty ? Text(subtitle) : null,
            ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.person_outline_rounded),
            title: Text(l10n.profileTitle),
            trailing: const Icon(Icons.chevron_right_rounded),
            onTap: () => context.push('/profile'),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.verified_user_outlined),
            title: Text(l10n.kycMenuLabel),
            trailing: const Icon(Icons.chevron_right_rounded),
            onTap: () => context.push('/kyc'),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.account_balance_wallet_outlined),
            title: Text(l10n.walletTitle),
            trailing: const Icon(Icons.chevron_right_rounded),
            onTap: () => context.push('/wallet'),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.receipt_long_rounded),
            title: Text(l10n.ordersTitle),
            trailing: const Icon(Icons.chevron_right_rounded),
            onTap: () => context.push('/orders'),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.privacy_tip_outlined),
            title: Text(l10n.privacyPolicyMenuLabel),
            trailing: const Icon(Icons.chevron_right_rounded),
            onTap: () => launchUrl(
              Uri.parse(AppConfig.privacyUrl),
              mode: LaunchMode.externalApplication,
            ),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.translate_rounded),
            title: Text(l10n.languageToggle),
            onTap: () =>
                ref.read(localeControllerProvider.notifier).toggle(),
            trailing: Text(isAr ? 'عربي' : 'EN'),
          ),
          SwitchListTile(
            secondary: const Icon(Icons.dark_mode_outlined),
            title: Text(l10n.darkMode),
            value: isDark,
            onChanged: (_) =>
                ref.read(themeControllerProvider.notifier).toggle(),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.logout_rounded, color: Color(0xFFFF4D6D)),
            title: Text(l10n.logout,
                style: const TextStyle(color: Color(0xFFFF4D6D))),
            onTap: () => ref.read(authControllerProvider.notifier).logout(),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.delete_forever_outlined,
                color: Color(0xFFFF4D6D)),
            title: Text(l10n.deleteAccountMenuLabel,
                style: const TextStyle(color: Color(0xFFFF4D6D))),
            onTap: () => showDeleteAccountSheet(context),
          ),
        ],
      ),
    );
  }
}

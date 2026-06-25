import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/auth_header.dart';
import '../../../../core/widgets/auth_switch_link.dart';
import '../../../../core/widgets/auth_text_field.dart';
import '../../../../core/widgets/country_code_box.dart';
import '../../../../core/widgets/gradient_heading.dart';
import '../../../../core/widgets/primary_cta.dart';
import '../controllers/auth_controller.dart';

/// Phone + password sign-in, on the same template as the sign-up flow. Client
/// side for now: a successful submit authenticates a demo session (no phone
/// auth backend yet — see [AuthController.completeDemoAuth]).
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _phone = TextEditingController();
  final _password = TextEditingController();

  String? _errorField; // 'phone' | 'password'
  String? _errorMessage;
  bool _loading = false;

  @override
  void dispose() {
    _phone.dispose();
    _password.dispose();
    super.dispose();
  }

  void _clearError() {
    if (_errorField != null) setState(() => _errorField = _errorMessage = null);
  }

  void _setError(String field, String message) {
    setState(() {
      _errorField = field;
      _errorMessage = message;
    });
  }

  void _submit() {
    if (_loading) return;
    final l10n = AppLocalizations.of(context);
    final digits = _phone.text.replaceAll(RegExp(r'\D'), '').length;
    if (digits < 7) return _setError('phone', l10n.invalidPhone);
    if (_password.text.isEmpty) {
      return _setError('password', l10n.passwordRequired);
    }
    setState(() {
      _errorField = _errorMessage = null;
      _loading = true;
    });
    Future.delayed(const Duration(milliseconds: 1600), () {
      if (!mounted) return;
      ref.read(authControllerProvider.notifier).completeDemoAuth();
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;

    return Scaffold(
      backgroundColor: colors.surface,
      body: SafeArea(
        child: Column(
          children: [
            const AuthHeader(),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GradientHeading(l10n.welcomeBackTitle),
                    const SizedBox(height: 10),
                    Text(
                      l10n.signInSubtitle,
                      style: TextStyle(
                        fontSize: 15.5,
                        height: 1.5,
                        color: colors.textDim,
                      ),
                    ),
                    const SizedBox(height: 28),
                    AuthTextField(
                      label: l10n.mobileNumberLabel,
                      controller: _phone,
                      hintText: l10n.phoneHint,
                      keyboardType: TextInputType.phone,
                      autofillHints: const [AutofillHints.telephoneNumber],
                      inputFormatters: [
                        FilteringTextInputFormatter.allow(RegExp(r'[0-9 ]')),
                        LengthLimitingTextInputFormatter(12),
                      ],
                      leading: const CountryCodeBox(),
                      errorText: _errorField == 'phone' ? _errorMessage : null,
                      onChanged: (_) => _clearError(),
                    ),
                    const SizedBox(height: 20),
                    AuthTextField(
                      label: l10n.passwordLabel,
                      controller: _password,
                      hintText: l10n.passwordHint,
                      obscureText: true,
                      textInputAction: TextInputAction.done,
                      errorText:
                          _errorField == 'password' ? _errorMessage : null,
                      onChanged: (_) => _clearError(),
                      onSubmitted: (_) => _submit(),
                    ),
                    const SizedBox(height: 30),
                    PrimaryCta(
                      label: l10n.signInButton,
                      onPressed: _submit,
                      loading: _loading,
                    ),
                    const SizedBox(height: 18),
                    AuthSwitchLink(
                      prefix: l10n.noAccountPrefix,
                      link: l10n.signUpButton,
                      onTap: () => context.go('/signup'),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

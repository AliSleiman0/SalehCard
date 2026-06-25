import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/auth_header.dart';
import '../../../../core/widgets/auth_switch_link.dart';
import '../../../../core/widgets/auth_text_field.dart';
import '../../../../core/widgets/country_code_box.dart';
import '../../../../core/widgets/gradient_heading.dart';
import '../../../../core/widgets/otp_input.dart';
import '../../../../core/widgets/primary_cta.dart';
import '../../../../core/widgets/step_dots.dart';
import '../controllers/auth_controller.dart';

enum _Step { phone, password, otp }

/// Phone-based sign-up flow modeled on the Claude Design "PhoneAuth" component:
/// phone entry → create password → OTP verification. Client-side flow for now;
/// the terminal verify authenticates a demo session (no phone/OTP backend yet).
class SignupScreen extends ConsumerStatefulWidget {
  const SignupScreen({super.key});

  @override
  ConsumerState<SignupScreen> createState() => _SignupScreenState();
}

class _SignupScreenState extends ConsumerState<SignupScreen> {
  final _phone = TextEditingController();
  final _password = TextEditingController();
  final _confirm = TextEditingController();
  String _otp = '';

  _Step _step = _Step.phone;
  String? _errorField; // 'phone' | 'password' | 'confirm' | 'otp'
  String? _errorMessage;
  bool _loading = false;

  @override
  void dispose() {
    _phone.dispose();
    _password.dispose();
    _confirm.dispose();
    super.dispose();
  }

  int get _digits => _phone.text.replaceAll(RegExp(r'\D'), '').length;

  void _clearError() {
    if (_errorField != null) setState(() => _errorField = _errorMessage = null);
  }

  void _setError(String field, String message) {
    setState(() {
      _errorField = field;
      _errorMessage = message;
    });
  }

  void _goBack() {
    setState(() {
      _errorField = _errorMessage = null;
      _step = _step == _Step.otp ? _Step.password : _Step.phone;
    });
  }

  void _submit() {
    if (_loading) return;
    final l10n = AppLocalizations.of(context);
    switch (_step) {
      case _Step.phone:
        if (_digits < 7) return _setError('phone', l10n.invalidPhone);
        setState(() => _step = _Step.password);
      case _Step.password:
        if (_password.text.length < 6) {
          return _setError('password', l10n.passwordTooShort);
        }
        if (_confirm.text != _password.text) {
          return _setError('confirm', l10n.passwordMismatch);
        }
        setState(() {
          _errorField = _errorMessage = null;
          _step = _Step.otp;
        });
      case _Step.otp:
        if (_otp.length < 6) return _setError('otp', l10n.otpIncomplete);
        setState(() {
          _errorField = _errorMessage = null;
          _loading = true;
        });
        Future.delayed(const Duration(milliseconds: 1600), () {
          if (!mounted) return;
          ref.read(authControllerProvider.notifier).completeDemoAuth();
        });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final isPhone = _step == _Step.phone;

    final String heading;
    final String subtitle;
    final String cta;
    switch (_step) {
      case _Step.phone:
        heading = l10n.getStartedTitle;
        subtitle = l10n.getStartedSubtitle;
        cta = l10n.continueButton;
      case _Step.password:
        heading = l10n.createPasswordTitle;
        subtitle = l10n.createPasswordSubtitle;
        cta = l10n.continueButton;
      case _Step.otp:
        final phone = _phone.text.isEmpty ? '70 123 456' : _phone.text;
        heading = l10n.verifyTitle;
        subtitle = l10n.otpSubtitle('+961 $phone');
        cta = l10n.verifyButton;
    }

    return Scaffold(
      backgroundColor: colors.surface,
      body: SafeArea(
        child: Column(
          children: [
            AuthHeader(onBack: isPhone ? null : _goBack),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    StepDots(count: 3, activeIndex: _step.index),
                    const SizedBox(height: 20),
                    GradientHeading(heading),
                    const SizedBox(height: 10),
                    Text(
                      subtitle,
                      style: TextStyle(
                        fontSize: 15.5,
                        height: 1.5,
                        color: colors.textDim,
                      ),
                    ),
                    const SizedBox(height: 28),
                    if (_step == _Step.phone) ..._phoneStep(l10n),
                    if (_step == _Step.password) ..._passwordStep(l10n),
                    if (_step == _Step.otp) ..._otpStep(l10n, colors),
                    const SizedBox(height: 30),
                    PrimaryCta(label: cta, onPressed: _submit, loading: _loading),
                    if (isPhone) ...[
                      const SizedBox(height: 18),
                      AuthSwitchLink(
                        prefix: l10n.haveAccountPrefix,
                        link: l10n.signInButton,
                        onTap: () => context.go('/login'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  List<Widget> _phoneStep(AppLocalizations l10n) {
    return [
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
      const SizedBox(height: 14),
      _LegalLine(prefix: l10n.termsPrefix, link: l10n.termsLink),
    ];
  }

  List<Widget> _passwordStep(AppLocalizations l10n) {
    return [
      AuthTextField(
        label: l10n.passwordLabel,
        controller: _password,
        hintText: l10n.newPasswordHint,
        obscureText: true,
        errorText: _errorField == 'password' ? _errorMessage : null,
        onChanged: (_) => _clearError(),
      ),
      const SizedBox(height: 20),
      AuthTextField(
        label: l10n.confirmPasswordLabel,
        controller: _confirm,
        hintText: l10n.confirmPasswordHint,
        obscureText: true,
        textInputAction: TextInputAction.done,
        errorText: _errorField == 'confirm' ? _errorMessage : null,
        onChanged: (_) => _clearError(),
        onSubmitted: (_) => _submit(),
      ),
    ];
  }

  List<Widget> _otpStep(AppLocalizations l10n, AppColors colors) {
    return [
      OtpInput(
        hasError: _errorField == 'otp',
        onChanged: (v) {
          _otp = v;
          _clearError();
        },
      ),
      if (_errorField == 'otp') ...[
        const SizedBox(height: 9),
        Text(
          _errorMessage!,
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w500,
            color: AppTokens.danger,
          ),
        ),
      ],
      const SizedBox(height: 16),
      Text.rich(
        TextSpan(
          children: [
            TextSpan(
              text: l10n.resendPrefix,
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: colors.textDim,
              ),
            ),
            TextSpan(
              text: l10n.resendLink,
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
                color: AppTokens.brand1,
              ),
            ),
          ],
        ),
      ),
    ];
  }
}

/// Legal text with an underlined magenta "terms and conditions" link.
class _LegalLine extends StatelessWidget {
  const _LegalLine({required this.prefix, required this.link});

  final String prefix;
  final String link;

  @override
  Widget build(BuildContext context) {
    final faint = context.colors.textFaint;
    return Padding(
      padding: const EdgeInsetsDirectional.only(start: 2, top: 0),
      child: Text.rich(
        TextSpan(
          children: [
            TextSpan(
              text: prefix,
              style: TextStyle(fontSize: 12.5, height: 1.55, color: faint),
            ),
            TextSpan(
              text: link,
              style: const TextStyle(
                fontSize: 12.5,
                color: AppTokens.brand2,
                decoration: TextDecoration.underline,
                decorationColor: AppTokens.brand2,
              ),
            ),
          ],
        ),
      ),
    );
  }
}


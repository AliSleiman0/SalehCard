import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../../auth/domain/entities/saved_player_id.dart';
import '../../../auth/domain/entities/user.dart';
import '../../../kyc/domain/entities/kyc.dart';
import '../../../kyc/presentation/providers.dart';
import '../providers.dart';

/// Profile screen (`/profile`). Watches [profileProvider] (`GET /users/me`).
///
/// Two states toggled by the AppBar action:
///  - **view**: read-only account header (neutral avatar + name/email), account
///    info rows, a verification (KYC) summary, and the saved player IDs as
///    labeled rows (or an [EmptyState] when none).
///  - **editing**: add a labeled player ID (label + value fields + button) and
///    remove existing ones (chips with a delete affordance), then a sticky "Save
///    changes" CTA that `PATCH`es `savedPlayerIds`. Name/email/role are read-only
///    here (name is set at signup); Role is hidden from customers.
class ProfileScreen extends ConsumerStatefulWidget {
  const ProfileScreen({super.key});

  @override
  ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  final _labelController = TextEditingController();
  final _valueController = TextEditingController();
  bool _editing = false;

  /// Working copy of the saved IDs while editing (seeded on enter-edit).
  List<SavedPlayerId> _draftIds = const [];

  @override
  void dispose() {
    _labelController.dispose();
    _valueController.dispose();
    super.dispose();
  }

  void _enterEdit(User user) {
    setState(() {
      _editing = true;
      _draftIds = List<SavedPlayerId>.from(user.savedPlayerIds);
    });
    ref.read(profileControllerProvider.notifier).reset();
  }

  void _cancelEdit() {
    _labelController.clear();
    _valueController.clear();
    setState(() => _editing = false);
    ref.read(profileControllerProvider.notifier).reset();
  }

  void _addId() {
    final label = _labelController.text.trim();
    final value = _valueController.text.trim();
    // Both fields are required; a duplicate value is silently ignored.
    if (label.isEmpty ||
        value.isEmpty ||
        _draftIds.any((e) => e.value == value)) {
      return;
    }
    setState(() {
      _draftIds = [..._draftIds, SavedPlayerId(label: label, value: value)];
      _labelController.clear();
      _valueController.clear();
    });
  }

  void _removeId(SavedPlayerId id) {
    setState(() => _draftIds = _draftIds.where((e) => e != id).toList());
  }

  Future<void> _save() async {
    final l10n = AppLocalizations.of(context);
    FocusScope.of(context).unfocus();
    final ok =
        await ref.read(profileControllerProvider.notifier).save(_draftIds);
    if (!mounted) return;
    if (ok) {
      setState(() => _editing = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.profileSaved)),
      );
    } else {
      final failure = ref.read(profileControllerProvider).failure;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(failure?.message.isNotEmpty == true
              ? failure!.message
              : l10n.loadFailed),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final profileAsync = ref.watch(profileProvider);
    final submitting = ref.watch(profileControllerProvider).submitting;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.profileTitle),
        actions: [
          profileAsync.maybeWhen(
            data: (user) => _editing
                ? TextButton(
                    onPressed: submitting ? null : _cancelEdit,
                    child: Text(l10n.cancel),
                  )
                : IconButton(
                    icon: const Icon(Icons.edit_outlined),
                    tooltip: l10n.editProfile,
                    onPressed: () => _enterEdit(user),
                  ),
            orElse: () => const SizedBox.shrink(),
          ),
        ],
      ),
      body: profileAsync.when(
        loading: () => const LoadingView(),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => ref.invalidate(profileProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (user) => Column(
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsetsDirectional.fromSTEB(20, 18, 20, 24),
                children: [
                  _AccountHeader(name: user.name, email: user.email),
                  const SizedBox(height: 22),
                  _SectionLabel(l10n.accountInfo),
                  const SizedBox(height: 10),
                  _InfoCard(
                    rows: [
                      if (user.name.isNotEmpty)
                        _InfoRow(label: l10n.nameLabel, value: user.name),
                      if (user.email.isNotEmpty)
                        _InfoRow(label: l10n.emailLabel, value: user.email),
                      if (user.phone != null && user.phone!.isNotEmpty)
                        _InfoRow(
                            label: l10n.mobileNumberLabel, value: user.phone!),
                      // Role is an internal concept — hidden from customers,
                      // shown only to resellers/admins who log into the app.
                      if (user.role != 'customer')
                        _InfoRow(label: l10n.roleLabel, value: user.role),
                      _InfoRow(
                        label: l10n.loyaltyPoints,
                        value: user.loyaltyPoints.toString(),
                        leading: Icons.star_rounded,
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),
                  // Verification (KYC) summary. Rendered from its own provider so
                  // a KYC fetch error/loading never breaks the profile page.
                  ..._verificationSection(l10n),
                  _SectionLabel(l10n.savedPlayerIds),
                  const SizedBox(height: 10),
                  if (_editing)
                    _PlayerIdEditor(
                      labelController: _labelController,
                      valueController: _valueController,
                      ids: _draftIds,
                      onAdd: _addId,
                      onRemove: _removeId,
                    )
                  else if (user.savedPlayerIds.isEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 12),
                      child: EmptyState(
                        icon: Icons.badge_outlined,
                        title: l10n.savedPlayerIdsEmptyTitle,
                        message: l10n.savedPlayerIdsEmptySub,
                        actionLabel: l10n.editProfile,
                        onAction: () => _enterEdit(user),
                      ),
                    )
                  else
                    _InfoCard(
                      rows: [
                        for (final id in user.savedPlayerIds)
                          _InfoRow(
                            label: id.label,
                            value: id.value,
                            leading: Icons.badge_outlined,
                          ),
                      ],
                    ),
                ],
              ),
            ),
            if (_editing)
              _SaveBar(
                label: l10n.saveChanges,
                submitting: submitting,
                colors: colors,
                onTap: _save,
              ),
          ],
        ),
      ),
    );
  }

  /// The verification section (label + card). Empty while the KYC status is
  /// erroring so the profile still renders; a small spinner while loading.
  List<Widget> _verificationSection(AppLocalizations l10n) {
    return ref.watch(kycProfileProvider).when(
          data: (profile) => [
            _SectionLabel(l10n.verificationTitle),
            const SizedBox(height: 10),
            _VerificationCard(
              profile: profile,
              onOpen: () => context.push('/kyc'),
            ),
            const SizedBox(height: 24),
          ],
          loading: () => [
            _SectionLabel(l10n.verificationTitle),
            const SizedBox(height: 10),
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 16),
              child: Center(child: AppSpinner(size: 22, stroke: 2.5)),
            ),
            const SizedBox(height: 24),
          ],
          error: (_, _) => const [],
        );
  }
}

class _AccountHeader extends StatelessWidget {
  const _AccountHeader({required this.name, required this.email});

  final String name;
  final String email;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final l10n = AppLocalizations.of(context);
    // Fall back to a friendly placeholder when the account has no name/email
    // (phone-only signups).
    final hasName = name.trim().isNotEmpty;
    final primary = hasName
        ? name.trim()
        : (email.isNotEmpty ? email : l10n.profileTitle);
    final hasSecondary = hasName && email.isNotEmpty;
    return Row(
      children: [
        Container(
          width: 64,
          height: 64,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            gradient: AppTokens.brandGradient,
            shape: BoxShape.circle,
          ),
          child: const Icon(Icons.person_rounded, color: Colors.white, size: 34),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                primary,
                style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w800,
                    color: colors.text),
              ),
              if (hasSecondary) ...[
                const SizedBox(height: 3),
                Text(
                  email,
                  style: TextStyle(fontSize: 13.5, color: colors.textDim),
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

/// Verification (KYC) summary card: a status badge, a tap target into the full
/// `/kyc` flow, and — once a submission exists — the submitted details.
class _VerificationCard extends StatelessWidget {
  const _VerificationCard({required this.profile, required this.onOpen});

  final KycProfile profile;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;

    final (badgeLabel, badgeColor) = switch (profile.status) {
      KycStatus.unverified => (l10n.kycBadgeUnverified, StatusBadge.neutral),
      KycStatus.pending => (l10n.kycBadgePending, StatusBadge.warning),
      KycStatus.verified => (l10n.kycBadgeVerified, StatusBadge.success),
      KycStatus.rejected => (l10n.kycBadgeRejected, StatusBadge.danger),
    };

    final s = profile.submission;

    return Container(
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rMd),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        children: [
          InkWell(
            onTap: onOpen,
            borderRadius: BorderRadius.circular(AppTokens.rMd),
            child: Padding(
              padding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
              child: Row(
                children: [
                  StatusBadge(label: badgeLabel, color: badgeColor),
                  const Spacer(),
                  if (profile.status == KycStatus.unverified ||
                      profile.status == KycStatus.rejected)
                    Text(
                      l10n.kycVerifyNowCta,
                      style: const TextStyle(
                          fontSize: 13.5,
                          fontWeight: FontWeight.w800,
                          color: AppTokens.brand1),
                    ),
                  Icon(Icons.chevron_right_rounded,
                      size: 20, color: colors.textFaint),
                ],
              ),
            ),
          ),
          if (s != null) ...[
            Divider(height: 1, color: colors.border),
            _InfoRow(label: l10n.kycFullNameLabel, value: s.fullName),
            Divider(height: 1, color: colors.border),
            _InfoRow(label: l10n.kycDobLabel, value: s.dateOfBirth),
            Divider(height: 1, color: colors.border),
            _InfoRow(label: l10n.kycPlaceOfBirthLabel, value: s.placeOfBirth),
            Divider(height: 1, color: colors.border),
            _InfoRow(
                label: l10n.kycPlaceOfResidenceLabel,
                value: s.placeOfResidence),
            Divider(height: 1, color: colors.border),
            _InfoRow(
                label: l10n.kycDocTypeLabel,
                value: _docLabel(l10n, s.documentType)),
            Divider(height: 1, color: colors.border),
            _InfoRow(
                label: l10n.kycDocNumberLabel, value: s.documentNumber),
          ],
        ],
      ),
    );
  }

  String _docLabel(AppLocalizations l10n, KycDocumentType t) => switch (t) {
        KycDocumentType.passport => l10n.kycDocPassport,
        KycDocumentType.idCard => l10n.kycDocIdCard,
        KycDocumentType.license => l10n.kycDocLicense,
      };
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: TextStyle(
          fontSize: 15,
          fontWeight: FontWeight.w800,
          color: context.colors.text),
    );
  }
}

class _InfoCard extends StatelessWidget {
  const _InfoCard({required this.rows});

  final List<_InfoRow> rows;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rMd),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        children: [
          for (var i = 0; i < rows.length; i++) ...[
            if (i > 0) Divider(height: 1, color: colors.border),
            rows[i],
          ],
        ],
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value, this.leading});

  final String label;
  final String value;
  final IconData? leading;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      child: Row(
        children: [
          if (leading != null) ...[
            Icon(leading, size: 18, color: colors.textFaint),
            const SizedBox(width: 12),
          ],
          if (label.isNotEmpty) ...[
            Text(
              label,
              style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim),
            ),
            const SizedBox(width: 12),
          ],
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.end,
              style: TextStyle(
                  fontSize: 14.5,
                  fontWeight: FontWeight.w700,
                  color: colors.text),
            ),
          ),
        ],
      ),
    );
  }
}

class _PlayerIdEditor extends StatelessWidget {
  const _PlayerIdEditor({
    required this.labelController,
    required this.valueController,
    required this.ids,
    required this.onAdd,
    required this.onRemove,
  });

  final TextEditingController labelController;
  final TextEditingController valueController;
  final List<SavedPlayerId> ids;
  final VoidCallback onAdd;
  final ValueChanged<SavedPlayerId> onRemove;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _EditorField(
          controller: labelController,
          hint: l10n.playerIdLabelHint,
          textInputAction: TextInputAction.next,
        ),
        const SizedBox(height: 10),
        Row(
          children: [
            Expanded(
              child: _EditorField(
                controller: valueController,
                hint: l10n.playerIdHint,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => onAdd(),
              ),
            ),
            const SizedBox(width: 10),
            // Add is enabled only when both the label and value are non-empty.
            AnimatedBuilder(
              animation: Listenable.merge([labelController, valueController]),
              builder: (context, _) {
                final enabled = labelController.text.trim().isNotEmpty &&
                    valueController.text.trim().isNotEmpty;
                return FilledButton(
                  onPressed: enabled ? onAdd : null,
                  style: FilledButton.styleFrom(
                    backgroundColor: AppTokens.cta,
                    foregroundColor: Colors.white,
                    disabledBackgroundColor:
                        AppTokens.cta.withValues(alpha: 0.5),
                    disabledForegroundColor: Colors.white,
                    minimumSize: const Size(72, 50),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(AppTokens.rMd),
                    ),
                  ),
                  child: Text(l10n.addPlayerId),
                );
              },
            ),
          ],
        ),
        if (ids.isNotEmpty) ...[
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final id in ids)
                Chip(
                  label: Text('${id.label}  ·  ${id.value}'),
                  labelStyle: TextStyle(
                      fontSize: 13.5,
                      fontWeight: FontWeight.w700,
                      color: colors.text),
                  backgroundColor: colors.surface,
                  side: BorderSide(color: colors.border),
                  deleteIcon: const Icon(Icons.close_rounded, size: 16),
                  onDeleted: () => onRemove(id),
                ),
            ],
          ),
        ],
      ],
    );
  }
}

/// A single outlined text field styled to match the profile editor.
class _EditorField extends StatelessWidget {
  const _EditorField({
    required this.controller,
    required this.hint,
    required this.textInputAction,
    this.onSubmitted,
  });

  final TextEditingController controller;
  final String hint;
  final TextInputAction textInputAction;
  final ValueChanged<String>? onSubmitted;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return TextField(
      controller: controller,
      onSubmitted: onSubmitted,
      textInputAction: textInputAction,
      style: TextStyle(fontSize: 15, color: colors.text),
      cursorColor: AppTokens.brand1,
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        hintText: hint,
        hintStyle: TextStyle(color: colors.textFaint),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        border: border,
        enabledBorder: border,
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
        ),
      ),
    );
  }
}

class _SaveBar extends StatelessWidget {
  const _SaveBar({
    required this.label,
    required this.submitting,
    required this.colors,
    required this.onTap,
  });

  final String label;
  final bool submitting;
  final AppColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: FilledButton(
        onPressed: submitting ? null : onTap,
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(54),
          backgroundColor: AppTokens.cta,
          disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.5),
          foregroundColor: Colors.white,
          disabledForegroundColor: Colors.white,
          shape: const StadiumBorder(),
          elevation: 8,
          shadowColor: AppTokens.cta.withValues(alpha: 0.3),
          textStyle: const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
        ),
        child: submitting
            ? const AppSpinner(color: Colors.white, size: 22, stroke: 2.5)
            : Text(label),
      ),
    );
  }
}

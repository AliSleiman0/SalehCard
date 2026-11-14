import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/format/date_time.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_back_button.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../data/complaint_repository.dart';
import '../../domain/complaint.dart';
import '../complaint_labels.dart';

/// One complaint as a chat thread (customer on the end side, support on the
/// start side) with a reply box, hidden once the complaint is closed. Reopened
/// live by a foreground complaint push (the push service invalidates
/// [complaintDetailProvider]).
class ComplaintDetailScreen extends ConsumerStatefulWidget {
  const ComplaintDetailScreen({super.key, required this.id});

  final String id;

  @override
  ConsumerState<ComplaintDetailScreen> createState() =>
      _ComplaintDetailScreenState();
}

class _ComplaintDetailScreenState extends ConsumerState<ComplaintDetailScreen> {
  final _reply = TextEditingController();
  bool _sending = false;

  @override
  void dispose() {
    _reply.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    final l10n = AppLocalizations.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final body = _reply.text.trim();
    if (body.isEmpty) return;
    setState(() => _sending = true);
    try {
      await ref.read(complaintRepositoryProvider).reply(widget.id, body);
      _reply.clear();
      ref.invalidate(complaintDetailProvider(widget.id));
      ref.invalidate(complaintsProvider);
    } catch (e) {
      final failure = mapError(e);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            failure.message.isNotEmpty
                ? failure.message
                : l10n.complaintSendFailed,
          ),
        ),
      );
      if (failure is ServerFailure && failure.code == 'CLOSED') {
        ref.invalidate(complaintDetailProvider(widget.id));
      }
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final async = ref.watch(complaintDetailProvider(widget.id));

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        // Reached by push from the list, or by go from a notification tap with
        // nothing beneath — fall back to the list rather than stranding the user.
        leading: const AppBackButton(fallback: '/complaints'),
        title: Text(async.value?.ref ?? l10n.complaintsTitle),
      ),
      body: async.when(
        loading: () => const LoadingView(),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.complaintsLoadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () =>
                      ref.invalidate(complaintDetailProvider(widget.id)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (c) {
          final (label, color) = complaintStatusLabel(c.status, l10n);
          return Column(
            children: [
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () async {
                    ref.invalidate(complaintDetailProvider(widget.id));
                    await ref.read(complaintDetailProvider(widget.id).future);
                  },
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
                    children: [
                      Wrap(
                        spacing: 8,
                        runSpacing: 6,
                        crossAxisAlignment: WrapCrossAlignment.center,
                        children: [
                          StatusBadge(label: label, color: color),
                          StatusBadge(
                            label: complaintCategoryLabel(c.category, l10n),
                            color: StatusBadge.neutral,
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      Text(
                        c.subject,
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w800,
                          color: colors.text,
                        ),
                      ),
                      if (c.orderId != null) ...[
                        const SizedBox(height: 4),
                        InkWell(
                          onTap: () => context.push('/orders/${c.orderId}'),
                          child: Text(
                            '${l10n.complaintLinkedOrder} #${c.orderId!.substring(c.orderId!.length - 8)}',
                            style: const TextStyle(
                              color: AppTokens.brand1,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                      ],
                      const SizedBox(height: 16),
                      for (final m in c.messages) _Bubble(m: m, l10n: l10n),
                      if (c.status == ComplaintStatus.resolved)
                        _Hint(text: l10n.complaintResolvedHint),
                      if (c.isClosed) _Hint(text: l10n.complaintClosedHint),
                    ],
                  ),
                ),
              ),
              if (!c.isClosed)
                SafeArea(
                  top: false,
                  child: Container(
                    padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
                    decoration: BoxDecoration(
                      color: colors.surface,
                      border: Border(top: BorderSide(color: colors.border)),
                    ),
                    child: Row(
                      children: [
                        Expanded(
                          child: TextField(
                            controller: _reply,
                            minLines: 1,
                            maxLines: 4,
                            maxLength: 2000,
                            decoration: InputDecoration(
                              hintText: l10n.complaintReplyHint,
                              counterText: '',
                              border: InputBorder.none,
                            ),
                          ),
                        ),
                        IconButton(
                          tooltip: l10n.complaintSend,
                          onPressed: _sending ? null : _send,
                          icon: _sending
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Icon(
                                  Icons.send_rounded,
                                  color: AppTokens.brand1,
                                ),
                        ),
                      ],
                    ),
                  ),
                ),
            ],
          );
        },
      ),
    );
  }
}

class _Hint extends StatelessWidget {
  const _Hint({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 8),
    child: Text(
      text,
      textAlign: TextAlign.center,
      style: TextStyle(fontSize: 12.5, color: context.colors.textFaint),
    ),
  );
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.m, required this.l10n});

  final ComplaintMessage m;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final mine = m.fromCustomer;
    return Align(
      alignment: mine
          ? AlignmentDirectional.centerEnd
          : AlignmentDirectional.centerStart,
      child: Container(
        margin: const EdgeInsets.only(bottom: 10),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.sizeOf(context).width * 0.8,
        ),
        padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 10),
        decoration: BoxDecoration(
          color: mine
              ? AppTokens.brand1.withValues(alpha: 0.16)
              : colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '${mine ? l10n.complaintYou : l10n.complaintSupport} · ${formatDateTime(m.createdAt)}',
              style: TextStyle(
                fontSize: 11.5,
                fontWeight: FontWeight.w700,
                color: colors.textFaint,
              ),
            ),
            const SizedBox(height: 4),
            Text(m.body, style: TextStyle(color: colors.text, height: 1.4)),
            if (m.photoUrl != null) ...[
              const SizedBox(height: 8),
              ClipRRect(
                borderRadius: BorderRadius.circular(AppTokens.rSm),
                child: Image.network(
                  m.photoUrl!,
                  width: 140,
                  height: 140,
                  fit: BoxFit.cover,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

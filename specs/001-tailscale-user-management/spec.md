# Feature Specification: Tailscale User Management

**Feature Branch**: `001-tailscale-user-management`  
**Created**: 2025-02-07  
**Status**: Draft  
**Input**: User description: "I want to be able to manage tailscale users, add, remove, disable, reenable and invite users."

### Reference model: invite and membership (GitHub provider alignment)

The user invite and membership lifecycle SHALL follow the same conceptual pattern as the [GitHub provider membership resource](https://registry.terraform.io/providers/integrations/github/latest/docs/resources/membership) ([source](https://github.com/integrations/terraform-provider-github)):

- **Single membership concept per identity**: One membership record per identity in the tailnet (or pending). There is no separate "invitation" resource from the administrator’s perspective; an invitation is the outcome of ensuring membership for an identity that is not yet in the tailnet.
- **Ensure membership = invite when needed**: When the administrator ensures that an identity has membership and that identity is not yet in the tailnet, the system sends an invitation (e.g. by email or link). If the identity is already a member or already has a pending invite, the operation is idempotent (no-op, success, state unchanged).
- **Same record for pending and active**: The same membership can be in state *pending* (invitation sent, not yet accepted), *active*, or *disabled*. The administrator can list and inspect memberships and see this state; they can update (e.g. role, disable/re-enable) or remove the membership.
- **Destroy = cancel invite or remove member**: When membership for an identity is removed (destroyed), the system either cancels the pending invitation (if not yet accepted) or removes the user from the tailnet (if already a member). Removal is idempotent if the identity is already no longer in the tailnet.
- **Optional "on destroy" behavior**: The system MAY support an option so that when membership is destroyed, the user is not removed but instead disabled (or downgraded), analogous to the GitHub provider’s `downgrade_on_destroy`.

## Clarifications

### Session 2025-02-07

- Q: When the administrator tries to remove or disable the last admin or the account owner, what behavior do we want? → A: Prevent: block remove/disable of the last admin or account owner and return a clear message.
- Q: When an administrator invites an identity that is already invited or already a member, what should happen? → A: Idempotent success: second invite for same identity is a no-op and returns success; state unchanged.
- Q: When two administrators act on the same user at the same time (e.g. one disables while another re-enables), how should conflicts be resolved? → A: Last write wins: the latest successful action determines the user's state; no conflict error.
- Design alignment: User invite and membership lifecycle follow the [GitHub provider membership](https://registry.terraform.io/providers/integrations/github/latest/docs/resources/membership) pattern: single membership per identity; ensure membership = invite when needed; destroy = cancel invite or remove member; optional disable/downgrade on destroy.
- Q: Should membership support an explicit role (e.g. admin vs member)? → A: Explicit role: membership has a role (e.g. "member", "admin"); default "member"; used for last-admin checks and optional "downgrade on destroy".
- Q: Who controls invite expiry and cancellation? → A: System-defined expiry; administrators can cancel a pending invite (e.g. by removing membership).
- Q: Should "disable/downgrade on destroy" be required or optional? → A: Keep as MAY (optional): the system MAY support the option; no requirement to implement it.
- Q: When an operation fails (e.g. backend unavailable), what should the administrator see? → A: Clear, actionable message indicating failure and, when possible, what to do (e.g. retry, check connectivity).
- Q: Should the spec include explicit out-of-scope items? → A: Add a short "Out of scope" subsection with 3–5 bullets.

### Session 2026-02-07

- Q: How should the system handle an ensure-membership request for an invalid identity (e.g. malformed email)? → A: Return a clear error; not idempotent.
- Q: How should the system represent invitations that have expired but are still listed by the backend? → A: Treat as pending until explicitly removed.
- Q: Which roles MUST the system support for membership? → A: Only "member" and "admin"; other roles are out of scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add or ensure membership (invite when needed) (Priority: P1)

An administrator wants to ensure an identity is in the tailnet (membership). If the identity is not yet a member and has no pending invite, the system sends an invitation (e.g. by email or link); the invitee can then accept and become an active member. If the identity is already a member or already has a pending invite, the operation is a no-op (idempotent).

**Why this priority**: Ensuring membership is the primary way to grow the tailnet; for new identities it results in sending an invite; for existing ones it is idempotent.

**Independent Test**: Can be fully tested by ensuring membership for a valid identity (e.g. email) and verifying an invite is sent when needed or state is unchanged when already present; delivers the ability to add/invite users.

**Acceptance Scenarios**:

1. **Given** the administrator has permission to manage users, **When** they ensure membership for a valid identity that is not in the tailnet, **Then** an invitation is created and sent (e.g. by email or link) and the invitee can join the tailnet upon acceptance.
2. **Given** a membership exists in pending state (invitation sent), **When** the invitee accepts, **Then** the membership becomes active and the invitee is a member of the tailnet.
3. **Given** the administrator ensures membership for an identity that is already in the tailnet (or already has a pending invite), **Then** the system responds with idempotent success and does not leave state inconsistent.
4. **Given** the administrator ensures membership for an invalid identity (e.g. malformed email), **Then** the system MUST return a clear validation error and MUST NOT create or modify any membership; the operation is not idempotent.
5. **Given** the administrator removes membership for an identity that is still pending (invite not yet accepted), **When** the removal is applied, **Then** the invitation is cancelled (invitee can no longer join via that invite).

---

### User Story 2 - Disable and Re-enable membership (Priority: P2)

An administrator needs to temporarily revoke a member’s access (e.g. leave of absence, security review) without permanently removing membership. Later, the same membership can be re-enabled and the user regains access.

**Why this priority**: Disable/re-enable supports compliance and temporary access control without losing user history or requiring a full remove-and-re-invite cycle.

**Independent Test**: Can be fully tested by disabling a user, confirming they lose access, then re-enabling and confirming they regain access.

**Acceptance Scenarios**:

1. **Given** a user is active in the tailnet, **When** the administrator disables that user, **Then** the user loses access to the tailnet and cannot use it until re-enabled.
2. **Given** a user is disabled, **When** the administrator re-enables that user, **Then** the user regains access to the tailnet with the same identity.
3. **Given** a user is already disabled, **When** the administrator requests disable again, **Then** the system behaves idempotently (no error, state unchanged).
4. **Given** a user is already active, **When** the administrator requests re-enable again, **Then** the system behaves idempotently.

---

### User Story 3 - Remove membership (Priority: P3)

An administrator needs to remove an identity’s membership from the tailnet (e.g. offboarding, contract end). If the identity had a pending invite, the invite is cancelled; if they were a member, they are removed and lose access.

**Why this priority**: Removal is necessary for offboarding and cleanup; it is used less frequently than invite or disable/re-enable.

**Independent Test**: Can be fully tested by removing a user and verifying they no longer have access and no longer appear as a member.

**Acceptance Scenarios**:

1. **Given** a user is in the tailnet (active or disabled), **When** the administrator removes that user, **Then** the user is no longer a member and loses all access to the tailnet.
2. **Given** the administrator removes a user, **When** the removal completes, **Then** the user’s devices and association with the tailnet are handled in a defined way (e.g. revoked, unlinked) so they cannot access the tailnet.
3. **Given** a user has already been removed, **When** the administrator requests removal again, **Then** the system responds in a safe, idempotent way (e.g. success or no-op, no unexpected error).

---

### User Story 4 - List and inspect memberships (Priority: P4)

An administrator needs to see all memberships in the tailnet and each membership’s state (e.g. pending, active, disabled) so they can decide whom to add, disable, re-enable, or remove.

**Why this priority**: Read-only visibility is required to manage users effectively but depends on at least one user existing (from invite or prior setup).

**Independent Test**: Can be fully tested by listing users and verifying the list reflects current members and their status.

**Acceptance Scenarios**:

1. **Given** the administrator has permission to view memberships, **When** they request a list of memberships, **Then** they see the set of memberships in the tailnet and each membership’s state (e.g. pending, active, disabled).
2. **Given** the list of memberships, **When** the administrator inspects a single membership, **Then** they see that identity, state, and role sufficient to support add/disable/re-enable/remove decisions.

---

### Edge Cases

- What happens when the administrator ensures membership for the same identity twice (e.g. duplicate ensure)? The system MUST treat the second operation as idempotent: no-op, return success, and leave state unchanged (one membership per identity).
- What happens when the administrator ensures membership for an invalid identity (e.g. malformed email, unsupported identifier)? The system MUST validate the identity, return a clear error, and MUST NOT create or modify any membership. Repeating the call with the same invalid identity MUST again error (not idempotent).
- What happens when the administrator disables or removes a user who has active sessions or devices? The system should revoke or disconnect access in a defined way so the user cannot continue using the tailnet.
- How does the system behave when the administrator tries to remove or disable the last admin or the account owner? The system MUST prevent remove and disable in this case and return a clear message so the tailnet cannot be left without an administrator.
- What happens when an invite expires or is cancelled before the invitee accepts? Invite expiry is defined by the system (e.g. fixed TTL). Administrators can cancel a pending invite (e.g. by removing membership). If a cancelled invite is no longer listed by the backend, the membership is treated as absent. If an expired invite is still listed by the backend, the system MUST treat the membership as `pending` until it is explicitly removed (by the administrator or by the backend); ensuring membership again for the same identity MUST be possible (no-op while still listed as pending; new invite once absent).
- When membership is removed (destroyed) before the invitee accepts, the invitation MUST be cancelled so the invitee cannot join via that invite.
- When two administrators act on the same user concurrently (e.g. one disables while another re-enables)? The system MUST apply last write wins: the latest successful action determines the user's state; no conflict error is required.
- When an operation fails (e.g. backend unavailable, rate limit)? The system MUST present a clear, actionable message indicating failure and, when possible, what to do (e.g. retry, check connectivity).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow authorized administrators to ensure membership for an identity in the tailnet (with an optional role from the supported set, default "member"); if the identity is not yet a member and has no pending invite, the system MUST send an invitation (e.g. by email or link).
- **FR-001a**: The system MUST validate the supplied identity (e.g. well-formed email) before any state change. If the identity is invalid, the system MUST return a clear, actionable validation error and MUST NOT create or modify any membership; this error path is not idempotent (a subsequent call with the same invalid identity MUST again error).
- **FR-002**: The system MUST allow authorized administrators to disable a user so that the user loses access to the tailnet until re-enabled.
- **FR-003**: The system MUST allow authorized administrators to re-enable a previously disabled user so that the user regains access.
- **FR-004**: The system MUST allow authorized administrators to remove membership for an identity: if the membership was pending (invite not yet accepted), the invitation MUST be cancelled; if the identity was a member, they MUST be removed and lose access.
- **FR-005**: The system MUST allow authorized administrators to list memberships in the tailnet and see each membership’s state (pending, active, disabled) and role (`member` or `admin`).
- **FR-005a**: The set of supported roles for membership MUST be exactly `{member, admin}`. Any other role value (including roles that may exist in the underlying backend) MUST be rejected with a clear validation error and is out of scope for this feature.
- **FR-006**: The system MUST treat duplicate or redundant operations on a valid identity in an idempotent way: ensure membership for same identity again (already member or pending), disable already disabled membership, or re-enable already active membership MUST be no-ops that return success and leave state unchanged. Idempotency does NOT apply to validation errors (see FR-001a).
- **FR-007**: The system MUST revoke or disconnect a user’s access when they are disabled or removed so they cannot continue using the tailnet.
- **FR-008**: The system MUST support at least the following states for a membership: pending (invitation sent, not yet accepted), active, and disabled; and the absence of membership (removed / no longer a member). An invitation that has expired but is still listed by the backend MUST be reported as `pending` until it is explicitly removed.
- **FR-009**: The system MUST prevent remove and disable of the last administrator (membership with admin role) or the account owner and MUST return a clear message when such an action is attempted.
- **FR-010**: When multiple administrators change the same membership's state concurrently, the system MUST apply last write wins (the latest successful action determines state); no conflict error is required.
- **FR-012**: When an operation fails (e.g. ensure membership, disable, remove, list), the system MUST present the administrator with a clear, actionable message indicating failure and, when possible, what to do (e.g. retry, check connectivity).
- **FR-011**: The system MAY support an option so that when a membership is removed (destroyed), the user is not removed from the tailnet but instead disabled or the role is downgraded to "member", analogous to the GitHub provider’s “downgrade on destroy” behavior.

### Key Entities

- **Membership**: The primary entity linking an identity to the tailnet. One membership per identity. Attributes include identity (e.g. email), state (pending, active, disabled), and role. Role MUST be exactly one of `{member, admin}` and default to `member`; it is used for last-admin protection and optional "downgrade on destroy". Any other role value is rejected as a validation error (see FR-005a). The identity itself MUST be valid (e.g. well-formed email) or the membership operation is rejected (see FR-001a). Pending = invitation sent, not yet accepted (including invitations that have expired but are still listed by the backend); active = member with access; disabled = member without access. When membership is removed, the identity is no longer in the tailnet (or invite is cancelled if still pending).
- **User**: The person (identity) who is or was part of the tailnet; represented by a membership when in the tailnet. A user may have zero or more devices once active.
- **Invitation**: The pending state of a membership before the invitee accepts; not a separate resource from the administrator’s perspective. Expiry is system-defined (e.g. fixed TTL). Administrators can cancel by removing membership. Attributes include the invited identity, creation time, and validity (expired or cancelled). While an invitation remains listed by the backend — even after its TTL has elapsed — the membership MUST be reported as `pending`; it transitions to absent only once the backend no longer lists it (cancelled or removed). When accepted, the membership becomes active.
- **Administrator**: An actor with permission to ensure membership (add/invite), disable, re-enable, remove, and list memberships within the tailnet.

### Assumptions

- “Tailnet” and “Tailscale” refer to the same networking/identity context (the private network and its user model).
- User identity is assumed to be something the backing system already supports (e.g. email); no new identity provider or auth method is in scope.
- Authorization (who can manage users) is defined by the existing tailnet/admin model; this feature assumes such roles exist and does not define them.
- Invitation delivery (e.g. email) and acceptance flow are provided by the underlying system; this feature assumes that ensuring membership for a new identity results in sending an invitation and that acceptance turns the membership state from pending to active. Invite expiry is system-defined; administrators can cancel a pending invite (e.g. by removing membership).

### Out of scope

- Bulk import of memberships (e.g. from a file or another system) in a single operation.
- Invitee-side acceptance flow: the act of accepting an invitation as the invitee (e.g. a dedicated "accept invite" action) is provided by the underlying system; this feature covers only the administrator’s ensure/list/disable/remove of membership.
- Custom per-invite expiry set by the administrator; expiry is system-defined only.
- Defining or changing how the tailnet determines who is an "administrator" or "account owner"; this feature assumes those roles exist.
- Auditing or history of membership changes (e.g. who invited whom, when); only current state is in scope.
- Roles other than `member` and `admin` (e.g. owner, billing-admin, IT-admin, network-admin, auditor, or any custom roles supported by the underlying backend); only `{member, admin}` are in scope for this feature.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An administrator can ensure membership for an identity and have the membership created (and an invite sent when needed) and visible (e.g. in a list or status) within a single operation.
- **SC-002**: A disabled user loses access to the tailnet; a re-enabled user regains access, with state changes reflected in a single operation each.
- **SC-003**: When membership is removed, the identity is no longer a member (or the invite is cancelled if pending); the outcome is visible to the administrator (e.g. membership no longer in list).
- **SC-004**: An administrator can list memberships and see correct state (e.g. pending, active, disabled) for each membership in the tailnet.
- **SC-005**: Duplicate or redundant operations (ensure membership for same identity again, disable already disabled) do not cause errors or inconsistent state; behavior is idempotent or clearly documented.

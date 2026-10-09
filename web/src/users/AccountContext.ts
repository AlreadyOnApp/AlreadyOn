// The signed-in server account (not a household profile). Owners can see and
// change everything; viewers only watch and manage their own profiles. The
// server enforces this on every request, so the UI only hides what would fail.
import { createContext, useContext } from 'react';

export type AccountRole = 'owner' | 'viewer';

export interface Account {
  username: string;
  role: AccountRole;
  // Only an owner may change this account's password.
  passwordLocked: boolean;
  // Updates the shown username after the account renamed itself.
  rename?: (username: string) => void;
}

export const AccountContext = createContext<Account>({ username: '', role: 'viewer', passwordLocked: false });

export function useAccount(): Account & { isOwner: boolean } {
  const account = useContext(AccountContext);
  return { ...account, isOwner: account.role === 'owner' };
}

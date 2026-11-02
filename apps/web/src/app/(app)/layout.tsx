import { UserService } from "@ledger-01/api";
import { ApiProvider } from "@/lib/api/providers";
import { orLogin, serverClient } from "@/lib/api/server";
import { AppNav } from "./app-nav";

// Every signed-in page renders inside this shell. The session is checked on the server first, so
// signed-out visitors never see an empty app.
export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const users = await serverClient(UserService);
  const me = await orLogin(users.getMe({}));

  return (
    <ApiProvider>
      <div className="mx-auto flex min-h-dvh max-w-content-max flex-col px-4 pb-28 sm:px-8 lg:px-12">
        <header className="sticky top-4 z-40 pt-4">
          <AppNav
            name={me.user?.name ?? ""}
            email={me.user?.email ?? ""}
            avatarUrl={me.user?.avatarUrl ?? ""}
          />
        </header>
        <main className="flex-1 pt-10 sm:pt-12">{children}</main>
      </div>
    </ApiProvider>
  );
}

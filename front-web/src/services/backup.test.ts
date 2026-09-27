import { beforeEach, describe, expect, it, vi } from "vitest";
import { downloadBackup, importBackup } from "./backup";

vi.stubGlobal("fetch", vi.fn());

describe("backup service", () => {
  beforeEach(() => vi.clearAllMocks());

  it("downloads a backup blob", async () => {
    const backupBlob = new Blob(["{}"], { type: "application/json" });
    (fetch as any).mockResolvedValueOnce({
      ok: true,
      blob: async () => backupBlob,
    });

    await expect(downloadBackup("token")).resolves.toBe(backupBlob);
    expect(fetch).toHaveBeenCalledWith("http://localhost:8080/backup", {
      headers: { Authorization: "Bearer token" },
    });
  });

  it("uploads the selected backup file", async () => {
    (fetch as any).mockResolvedValueOnce({ ok: true });
    const backupFile = new File(['{"version":1}'], "backup.json", {
      type: "application/json",
    });

    await expect(importBackup("token", backupFile)).resolves.toBeUndefined();
    expect(fetch).toHaveBeenCalledWith("http://localhost:8080/backup", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: '{"version":1}',
    });
  });

  it("reports failed backup requests", async () => {
    (fetch as any).mockResolvedValueOnce({ ok: false, status: 500 });

    await expect(downloadBackup("token")).rejects.toThrow(
      "Backup export failed (500)",
    );
  });
});

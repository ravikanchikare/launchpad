import { toast } from "@/components/ui/toast"
import {
  hasNativeUpdateDialogs,
  nativeBridge,
} from "@/lib/native"
import type { UpdateStatus } from "@/types"

export function latestBuildMessage(currentVersion?: string) {
  if (currentVersion) {
    return `You are already on the latest build (${currentVersion}).`
  }
  return "You are already on the latest build."
}

export async function showUpdateAlert(title: string, message: string) {
  if (hasNativeUpdateDialogs()) {
    await nativeBridge.presentUpdateAlert(title, message)
    return
  }
  toast.add({ title: message, type: "neutral" })
}

export async function showUpdateError(message: string) {
  if (hasNativeUpdateDialogs()) {
    await nativeBridge.presentUpdateAlert("HarnezPad Updates", message)
    return
  }
  toast.add({ title: message, type: "error" })
}

export async function confirmUpdateInstall(version: string) {
  const message = `HarnezPad ${version} is ready to install. Restart now?`
  if (hasNativeUpdateDialogs()) {
    const result = await nativeBridge.presentUpdateConfirm("HarnezPad Updates", message)
    return result.confirmed
  }
  return null
}

export function formatUpdateError(message: string) {
  const lower = message.toLowerCase()
  if (lower.includes("self-update is unavailable")) {
    return "Updates are not available in this build."
  }
  if (lower.includes("no verified update is ready")) {
    return "No update is ready to install yet."
  }
  if (lower.includes("update checking is not available")) {
    return "Update checking is not available yet."
  }
  if (lower.includes("downloaded update sha-256")) {
    return "The downloaded update could not be verified. Try checking for updates again."
  }
  if (lower.includes("previous update is awaiting cleanup")) {
    return "Restart HarnezPad before installing another update."
  }
  if (
    lower.includes("could not be downloaded") ||
    lower.includes("check for update:") ||
    lower.includes("download update:")
  ) {
    return "Could not download the update. Check your connection and try again."
  }
  return message
}

export function updateReady(status: UpdateStatus | null) {
  return Boolean(
    status &&
      !status.error &&
      status.available &&
      status.downloaded &&
      status.update?.version
  )
}

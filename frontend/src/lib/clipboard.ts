import { toast } from "@/components/ui/toast"

export async function copyText(text: string) {
  try {
    if (window.zero)
      await window.zero.invoke("native-sdk.clipboard.writeText", { text })
    else await navigator.clipboard.writeText(text)
    toast.add({ title: "Copied to clipboard", type: "success" })
  } catch {
    toast.add({ title: "Couldn't copy to clipboard", type: "error" })
  }
}

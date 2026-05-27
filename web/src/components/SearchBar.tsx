import { createSignal } from "solid-js";

interface Props {
  onSearch: (query: string) => void;
  placeholder?: string;
}

export default function SearchBar(props: Props) {
  const [value, setValue] = createSignal("");

  const submit = (e: Event) => {
    e.preventDefault();
    props.onSearch(value());
  };

  return (
    <form class="search-bar" onSubmit={submit}>
      <input
        type="text"
        value={value()}
        onInput={(e) => setValue(e.currentTarget.value)}
        placeholder={props.placeholder ?? "Search sessions…"}
      />
      <button type="submit">Search</button>
    </form>
  );
}
